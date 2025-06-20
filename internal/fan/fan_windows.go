//go:build windows

package fan

import (
	"context"
	"fmt"

	"github.com/StackExchange/wmi"
)

// WindowsController implements fan control for Windows
type WindowsController struct {
	fans []WindowsFanInfo
}

// WindowsFanInfo represents a Windows fan device
type WindowsFanInfo struct {
	DeviceID     string
	Name         string
	RPM          int
	MaxRPM       int
	Speed        int
	Controllable bool
	WMIPath      string
	Source       string
}

// Win32_Fan represents WMI Win32_Fan class
type Win32_Fan struct {
	DeviceID                string
	Name                    string
	Description             string
	DesiredSpeed            uint64
	VariableSpeed           bool
	ActiveCooling           bool
	Status                  string
	SystemCreationClassName string
	SystemName              string
}

// Win32_TemperatureProbe represents WMI temperature probe
type Win32_TemperatureProbe struct {
	DeviceID       string
	Name           string
	Description    string
	CurrentReading int32
	NormalMax      int32
	MaxReadable    int32
	MinReadable    int32
	Status         string
}

// Win32_BaseBoard represents motherboard WMI class
type Win32_BaseBoard struct {
	Manufacturer string
	Product      string
	Version      string
}

// Win32_OperatingSystem represents OS WMI class
type Win32_OperatingSystem struct {
	Caption     string
	Version     string
	BuildNumber string
}

// newPlatformController creates a new Windows fan controller
func newPlatformController() Controller {
	controller := &WindowsController{}
	controller.discoverFans()
	return controller
}

// discoverFans discovers available fans using WMI
func (c *WindowsController) discoverFans() {
	// Method 1: Try WMI Win32_Fan (limited but standard)
	c.discoverWMIFans()

	// Method 2: Try temperature probes (may have associated fans)
	c.discoverTemperatureProbes()

	// If no fans found, add dummy fans for basic monitoring
	if len(c.fans) == 0 {
		c.addDummyFans()
	}
}

// discoverWMIFans discovers fans using WMI Win32_Fan
func (c *WindowsController) discoverWMIFans() {
	var fans []Win32_Fan
	err := wmi.Query("SELECT DeviceID, Name, Description, DesiredSpeed, VariableSpeed FROM Win32_Fan", &fans)
	if err != nil {
		return
	}

	for _, wmiFan := range fans {
		fanInfo := WindowsFanInfo{
			DeviceID:     wmiFan.DeviceID,
			Name:         wmiFan.Name,
			RPM:          int(wmiFan.DesiredSpeed),
			MaxRPM:       0,
			Speed:        0,
			Controllable: wmiFan.VariableSpeed,
			WMIPath:      fmt.Sprintf("Win32_Fan.DeviceID='%s'", wmiFan.DeviceID),
			Source:       "WMI",
		}

		if fanInfo.Name == "" {
			fanInfo.Name = fmt.Sprintf("Fan %s", wmiFan.DeviceID)
		}

		c.fans = append(c.fans, fanInfo)
	}
}

// discoverTemperatureProbes discovers temperature probes that may have associated fans
func (c *WindowsController) discoverTemperatureProbes() {
	var probes []Win32_TemperatureProbe
	err := wmi.Query("SELECT DeviceID, Name, Description FROM Win32_TemperatureProbe", &probes)
	if err != nil {
		return
	}

	for i, probe := range probes {
		fanInfo := WindowsFanInfo{
			DeviceID:     fmt.Sprintf("probe_fan_%d", i),
			Name:         fmt.Sprintf("Fan for %s", probe.Name),
			RPM:          0,
			MaxRPM:       0,
			Speed:        0,
			Controllable: false,
			WMIPath:      "",
			Source:       "TemperatureProbe",
		}

		if probe.Name == "" {
			fanInfo.Name = fmt.Sprintf("System Fan %d (Probe)", i+1)
		}

		c.fans = append(c.fans, fanInfo)
	}
}

// addDummyFans adds dummy fans for basic monitoring when no real fans are detected
func (c *WindowsController) addDummyFans() {
	dummyFans := []WindowsFanInfo{
		{
			DeviceID:     "cpu_fan",
			Name:         "CPU Fan (ID:0)",
			RPM:          0,
			MaxRPM:       0,
			Speed:        0,
			Controllable: false,
			WMIPath:      "",
			Source:       "Dummy",
		},
		{
			DeviceID:     "system_fan_1",
			Name:         "System Fan 1 (ID:1)",
			RPM:          0,
			MaxRPM:       0,
			Speed:        0,
			Controllable: false,
			WMIPath:      "",
			Source:       "Dummy",
		},
		{
			DeviceID:     "system_fan_2",
			Name:         "System Fan 2 (ID:2)",
			RPM:          0,
			MaxRPM:       0,
			Speed:        0,
			Controllable: false,
			WMIPath:      "",
			Source:       "Dummy",
		},
	}

	c.fans = append(c.fans, dummyFans...)
}

// GetFans returns all discovered fans
func (c *WindowsController) GetFans(ctx context.Context) ([]*Info, error) {
	var fans []*Info

	for _, fanInfo := range c.fans {
		info := &Info{
			Name:   fanInfo.Name,
			RPM:    fanInfo.RPM,
			Speed:  fanInfo.Speed,
			MaxRPM: fanInfo.MaxRPM,
		}
		fans = append(fans, info)
	}

	return fans, nil
}

// GetSettings returns fan settings for a specific fan
func (c *WindowsController) GetSettings(ctx context.Context, fanID int) (*Settings, error) {
	if fanID < 0 || fanID >= len(c.fans) {
		return nil, fmt.Errorf("fan ID %d out of range (0-%d)", fanID, len(c.fans)-1)
	}

	fan := c.fans[fanID]

	// Default to auto mode for Windows fans
	settings := &Settings{
		Mode:       ModeAuto,
		FixedSpeed: fan.Speed,
		Curve:      nil,
	}

	return settings, nil
}

// SetSettings applies fan settings
func (c *WindowsController) SetSettings(ctx context.Context, fanID int, settings *Settings) error {
	if fanID < 0 || fanID >= len(c.fans) {
		return fmt.Errorf("fan ID %d out of range (0-%d)", fanID, len(c.fans)-1)
	}

	fan := c.fans[fanID]

	if !fan.Controllable {
		return fmt.Errorf("fan %d (%s) is not controllable via software", fanID, fan.Name)
	}

	switch settings.Mode {
	case ModeFixed:
		return c.setFixedSpeed(fanID, settings.FixedSpeed)
	case ModeAuto:
		return c.setAutoMode(fanID)
	case ModeCurve:
		return fmt.Errorf("fan curve mode not supported on Windows - use motherboard software or third-party tools")
	default:
		return fmt.Errorf("unsupported fan mode: %s", settings.Mode)
	}
}

// setFixedSpeed sets a fixed fan speed
func (c *WindowsController) setFixedSpeed(fanID int, speedPercent int) error {
	if speedPercent < 0 || speedPercent > 100 {
		return fmt.Errorf("fan speed must be between 0 and 100 percent")
	}

	fan := c.fans[fanID]

	// Windows fan control through WMI is very limited
	// Most motherboards require vendor-specific software or BIOS settings
	return fmt.Errorf("direct fan control not supported for %s - use motherboard software (MSI Center, ASUS AI Suite, etc.) or third-party tools (SpeedFan, FanControl)", fan.Name)
}

// setAutoMode sets fan to automatic mode
func (c *WindowsController) setAutoMode(fanID int) error {
	fan := c.fans[fanID]

	// Auto mode would typically reset to BIOS/motherboard control
	return fmt.Errorf("automatic fan mode switching not supported for %s - use BIOS settings or motherboard software", fan.Name)
}
