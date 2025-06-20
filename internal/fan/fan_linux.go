//go:build linux

package fan

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LinuxController implements fan control for Linux
type LinuxController struct {
	pwmPaths    []string
	fanPaths    []string
	tempPaths   []string
	curveStates map[int]*curveState
}

// curveState holds the state for a fan running in curve mode
type curveState struct {
	curve       []CurvePoint
	lastTemp    int
	isActive    bool
	stopChannel chan bool
}

// newPlatformController creates a new Linux fan controller
func newPlatformController() Controller {
	controller := &LinuxController{
		curveStates: make(map[int]*curveState),
	}
	controller.discoverFans()
	return controller
}

// discoverFans finds available PWM and fan monitoring paths
func (c *LinuxController) discoverFans() {
	// Look for PWM control files
	pwmGlob := "/sys/class/hwmon/hwmon*/pwm*"
	if matches, err := filepath.Glob(pwmGlob); err == nil {
		for _, match := range matches {
			// Skip pwm files that are not the main control files
			if strings.HasSuffix(match, "_enable") ||
				strings.HasSuffix(match, "_mode") ||
				strings.Contains(match, "_auto_point") ||
				strings.Contains(match, "_temp") ||
				strings.Contains(match, "_crit") ||
				strings.Contains(match, "_floor") ||
				strings.Contains(match, "_start") ||
				strings.Contains(match, "_step") ||
				strings.Contains(match, "_stop") ||
				strings.Contains(match, "_target") ||
				strings.Contains(match, "_sel") ||
				strings.Contains(match, "_tolerance") ||
				strings.Contains(match, "_weight") {
				continue
			}

			// Only add base pwm files (e.g., pwm1, pwm2)
			baseName := filepath.Base(match)
			if strings.HasPrefix(baseName, "pwm") && !strings.Contains(baseName, "_") {
				if _, err := os.Stat(match); err == nil {
					c.pwmPaths = append(c.pwmPaths, match)
				}
			}
		}
	}

	// Look for fan input files
	fanGlob := "/sys/class/hwmon/hwmon*/fan*_input"
	if matches, err := filepath.Glob(fanGlob); err == nil {
		c.fanPaths = matches
	}

	// Look for temperature sensor files
	tempGlob := "/sys/class/hwmon/hwmon*/temp*_input"
	if matches, err := filepath.Glob(tempGlob); err == nil {
		c.tempPaths = matches
	}
}

// GetFans returns fan information
func (c *LinuxController) GetFans(ctx context.Context) ([]*Info, error) {
	var fans []*Info

	// First, try to use lm-sensors to get fan info
	cmd := exec.CommandContext(ctx, "sensors")
	output, err := cmd.Output()
	sensorsAvailable := err == nil

	if sensorsAvailable {
		lines := strings.Split(string(output), "\n")
		fanIndex := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "fan") && strings.Contains(line, "RPM") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					rpmStr := parts[1]
					if rpm, err := strconv.Atoi(rpmStr); err == nil {
						fan := &Info{
							Name:   parts[0],
							RPM:    rpm,
							Speed:  0, // Will be populated below
							MaxRPM: 0, // Will be estimated below
						}

						// Try to get PWM speed percentage for this fan
						if fanIndex < len(c.pwmPaths) {
							if speed := c.getPWMSpeedPercent(c.pwmPaths[fanIndex]); speed >= 0 {
								fan.Speed = speed
							}
							// Estimate max RPM based on current RPM and speed percentage
							if rpm > 0 && fan.Speed > 0 {
								estimatedMaxRPM := (rpm * 100) / fan.Speed
								fan.MaxRPM = estimatedMaxRPM
							}
						}

						fans = append(fans, fan)
						fanIndex++
					}
				}
			}
		}
	}

	// If no fans found via sensors or sensors not available,
	// create entries based on available PWM controls and try to read RPM directly
	if len(fans) == 0 {
		for i, pwmPath := range c.pwmPaths {
			// Extract PWM number from path (e.g., /sys/class/hwmon/hwmon2/pwm1 -> "pwm1")
			pwmName := filepath.Base(pwmPath)
			speed := c.getPWMSpeedPercent(pwmPath)

			// Try to read RPM directly from fan input files
			rpm := c.getFanRPMDirect(i)

			fan := &Info{
				Name:   fmt.Sprintf("%s (ID:%d)", pwmName, i),
				RPM:    rpm,
				Speed:  speed,
				MaxRPM: 0, // Cannot estimate without both RPM and speed
			}

			// Estimate max RPM if we have both values
			if rpm > 0 && speed > 0 {
				estimatedMaxRPM := (rpm * 100) / speed
				fan.MaxRPM = estimatedMaxRPM
			}

			fans = append(fans, fan)
		}
	}

	return fans, nil
}

// getFanRPMDirect attempts to read RPM directly from fan input files
func (c *LinuxController) getFanRPMDirect(fanIndex int) int {
	// Try to find corresponding fan input file
	for _, fanPath := range c.fanPaths {
		// Extract fan number from path (e.g., fan1_input -> 1)
		baseName := filepath.Base(fanPath)
		if strings.HasPrefix(baseName, fmt.Sprintf("fan%d_", fanIndex+1)) {
			data, err := ioutil.ReadFile(fanPath)
			if err != nil {
				continue
			}

			rpm, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				continue
			}

			return rpm
		}
	}
	return 0
}

// getPWMSpeedPercent reads the current PWM value and converts to percentage
func (c *LinuxController) getPWMSpeedPercent(pwmPath string) int {
	data, err := ioutil.ReadFile(pwmPath)
	if err != nil {
		return -1
	}

	pwmVal, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1
	}

	// Convert PWM value (0-255) to percentage (0-100)
	percentage := (pwmVal * 100) / 255
	return percentage
}

// GetSettings returns current fan settings
func (c *LinuxController) GetSettings(ctx context.Context, fanID int) (*Settings, error) {
	if fanID >= len(c.pwmPaths) {
		return nil, fmt.Errorf("fan ID %d not found", fanID)
	}

	// Check if fan is in curve mode
	if state, exists := c.curveStates[fanID]; exists && state.isActive {
		return &Settings{
			Mode:  ModeCurve,
			Curve: state.curve,
		}, nil
	}

	pwmPath := c.pwmPaths[fanID]
	enablePath := pwmPath + "_enable"

	// Check if PWM is enabled
	enableData, err := ioutil.ReadFile(enablePath)
	if err != nil {
		return &Settings{Mode: ModeAuto}, nil // Assume auto if can't read
	}

	enableVal := strings.TrimSpace(string(enableData))

	// Read current PWM value
	pwmData, err := ioutil.ReadFile(pwmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PWM value: %w", err)
	}

	pwmVal, err := strconv.Atoi(strings.TrimSpace(string(pwmData)))
	if err != nil {
		return nil, fmt.Errorf("invalid PWM value: %w", err)
	}

	settings := &Settings{}

	switch enableVal {
	case "1": // Manual/fixed mode
		settings.Mode = ModeFixed
		settings.FixedSpeed = (pwmVal * 100) / 255
	case "2": // Automatic mode
		settings.Mode = ModeAuto
	default:
		settings.Mode = ModeAuto
	}

	return settings, nil
}

// SetSettings applies fan settings
func (c *LinuxController) SetSettings(ctx context.Context, fanID int, settings *Settings) error {
	if fanID >= len(c.pwmPaths) {
		return fmt.Errorf("fan ID %d not found", fanID)
	}

	pwmPath := c.pwmPaths[fanID]
	enablePath := pwmPath + "_enable"

	// Check if we have write permissions
	if err := c.checkWritePermissions(pwmPath, enablePath); err != nil {
		return fmt.Errorf("insufficient permissions for fan control: %w", err)
	}

	switch settings.Mode {
	case ModeFixed:
		// Set to manual mode
		if err := ioutil.WriteFile(enablePath, []byte("1"), 0644); err != nil {
			return fmt.Errorf("failed to set manual mode: %w", err)
		}

		// Convert percentage to PWM value (0-255)
		pwmVal := (settings.FixedSpeed * 255) / 100
		if pwmVal > 255 {
			pwmVal = 255
		}
		if pwmVal < 0 {
			pwmVal = 0
		}

		if err := ioutil.WriteFile(pwmPath, []byte(fmt.Sprintf("%d", pwmVal)), 0644); err != nil {
			return fmt.Errorf("failed to set PWM value: %w", err)
		}

	case ModeAuto:
		// Set to automatic mode
		if err := ioutil.WriteFile(enablePath, []byte("2"), 0644); err != nil {
			return fmt.Errorf("failed to set automatic mode: %w", err)
		}

	case ModeCurve:
		// Stop any existing curve control for this fan
		c.stopFanCurve(fanID)

		// Validate curve points
		if len(settings.Curve) < 2 {
			return fmt.Errorf("fan curve must have at least 2 points")
		}

		// Sort curve points by temperature
		sortedCurve := make([]CurvePoint, len(settings.Curve))
		copy(sortedCurve, settings.Curve)
		sort.Slice(sortedCurve, func(i, j int) bool {
			return sortedCurve[i].Temperature < sortedCurve[j].Temperature
		})

		// Validate curve points
		for _, point := range sortedCurve {
			if point.Temperature < 0 || point.Temperature > 100 {
				return fmt.Errorf("temperature must be between 0 and 100°C")
			}
			if point.FanSpeed < 0 || point.FanSpeed > 100 {
				return fmt.Errorf("fan speed must be between 0 and 100%%")
			}
		}

		// Set to manual mode first
		if err := ioutil.WriteFile(enablePath, []byte("1"), 0644); err != nil {
			return fmt.Errorf("failed to set manual mode for curve: %w", err)
		}

		// Start curve control
		c.startFanCurve(fanID, sortedCurve)

	default:
		return fmt.Errorf("unsupported fan mode: %s", settings.Mode)
	}

	return nil
}

func (c *LinuxController) checkWritePermissions(paths ...string) error {
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("path %s not accessible: %w", path, err)
		}

		// Try to open for writing
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("no write permission for %s (try running as root or add user to appropriate group): %w", path, err)
		}
		file.Close()
	}
	return nil
}

// startFanCurve starts a goroutine to control the fan based on temperature curve
func (c *LinuxController) startFanCurve(fanID int, curve []CurvePoint) {
	state := &curveState{
		curve:       curve,
		isActive:    true,
		stopChannel: make(chan bool),
	}
	c.curveStates[fanID] = state

	go func() {
		ticker := time.NewTicker(2 * time.Second) // Update every 2 seconds
		defer ticker.Stop()

		for {
			select {
			case <-state.stopChannel:
				return
			case <-ticker.C:
				temp := c.getCurrentTemperature()
				if temp > 0 {
					fanSpeed := c.interpolateFanSpeed(curve, temp)
					c.setPWMSpeed(fanID, fanSpeed)
					state.lastTemp = temp
				}
			}
		}
	}()
}

// stopFanCurve stops the curve control for a fan
func (c *LinuxController) stopFanCurve(fanID int) {
	if state, exists := c.curveStates[fanID]; exists && state.isActive {
		state.isActive = false
		close(state.stopChannel)
		delete(c.curveStates, fanID)
	}
}

// getCurrentTemperature reads the current CPU temperature
func (c *LinuxController) getCurrentTemperature() int {
	// Try to get CPU temperature from common paths
	tempPaths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}

	// Also try discovered temperature paths
	tempPaths = append(tempPaths, c.tempPaths...)

	for _, path := range tempPaths {
		if data, err := ioutil.ReadFile(path); err == nil {
			if temp, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				// Convert millidegrees to degrees if necessary
				if temp > 1000 {
					temp = temp / 1000
				}
				if temp > 0 && temp < 120 { // Reasonable temperature range
					return temp
				}
			}
		}
	}

	// Fallback: try sensors command
	if cmd := exec.Command("sensors"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Core 0") || strings.Contains(line, "CPU") {
					if strings.Contains(line, "°C") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.Contains(part, "°C") {
								tempStr := strings.TrimSuffix(strings.TrimPrefix(part, "+"), "°C")
								if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
									return int(temp)
								}
							}
						}
					}
				}
			}
		}
	}

	return 0 // Could not read temperature
}

// interpolateFanSpeed calculates fan speed based on temperature and curve
func (c *LinuxController) interpolateFanSpeed(curve []CurvePoint, temp int) int {
	// If temperature is below the first point, use first point's speed
	if temp <= curve[0].Temperature {
		return curve[0].FanSpeed
	}

	// If temperature is above the last point, use last point's speed
	if temp >= curve[len(curve)-1].Temperature {
		return curve[len(curve)-1].FanSpeed
	}

	// Find the two points to interpolate between
	for i := 0; i < len(curve)-1; i++ {
		if temp >= curve[i].Temperature && temp <= curve[i+1].Temperature {
			// Linear interpolation
			tempRange := curve[i+1].Temperature - curve[i].Temperature
			speedRange := curve[i+1].FanSpeed - curve[i].FanSpeed
			tempDiff := temp - curve[i].Temperature

			interpolatedSpeed := curve[i].FanSpeed + (speedRange*tempDiff)/tempRange
			return interpolatedSpeed
		}
	}

	// Fallback (should not reach here)
	return curve[len(curve)-1].FanSpeed
}

// setPWMSpeed sets the PWM speed for a fan
func (c *LinuxController) setPWMSpeed(fanID int, speedPercent int) error {
	if fanID >= len(c.pwmPaths) {
		return fmt.Errorf("fan ID %d not found", fanID)
	}

	// Clamp speed to valid range
	if speedPercent < 0 {
		speedPercent = 0
	}
	if speedPercent > 100 {
		speedPercent = 100
	}

	// Convert percentage to PWM value (0-255)
	pwmVal := (speedPercent * 255) / 100

	pwmPath := c.pwmPaths[fanID]
	return ioutil.WriteFile(pwmPath, []byte(fmt.Sprintf("%d", pwmVal)), 0644)
}
