//go:build windows

package overclock

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// WindowsController implements overclocking control for Windows
type WindowsController struct {
	profilesDir string
}

// newPlatformController creates a new Windows overclocking controller
func newPlatformController() Controller {
	homeDir, _ := os.UserHomeDir()
	profilesDir := filepath.Join(homeDir, "AppData", "Local", "picohwmon", "profiles")

	// Create profiles directory if it doesn't exist
	os.MkdirAll(profilesDir, 0755)

	return &WindowsController{
		profilesDir: profilesDir,
	}
}

// GetSettings returns current overclocking settings
func (c *WindowsController) GetSettings(ctx context.Context, deviceID int) (*Settings, error) {
	vendor, err := c.detectGPUVendor(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to detect GPU vendor: %w", err)
	}

	switch vendor {
	case "nvidia":
		return c.getNvidiaSettings(ctx, deviceID)
	case "amd":
		return c.getAMDSettings(ctx, deviceID)
	default:
		return &Settings{DeviceID: deviceID}, fmt.Errorf("unsupported GPU vendor: %s", vendor)
	}
}

// SetSettings applies overclocking settings
func (c *WindowsController) SetSettings(ctx context.Context, settings *Settings) error {
	if err := c.validateSettings(settings); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	vendor, err := c.detectGPUVendor(ctx, settings.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to detect GPU vendor: %w", err)
	}

	var applyErr error
	switch vendor {
	case "nvidia":
		applyErr = c.setNvidiaSettings(ctx, settings)
	case "amd":
		applyErr = c.setAMDSettings(ctx, settings)
	default:
		applyErr = fmt.Errorf("unsupported GPU vendor: %s", vendor)
	}

	if applyErr != nil {
		return applyErr
	}

	// Save current settings to a temporary profile
	tempProfile := &Profile{
		Name:     "_current",
		Settings: settings,
	}

	profilePath := filepath.Join(c.profilesDir, "_current.json")
	data, err := json.MarshalIndent(tempProfile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := ioutil.WriteFile(profilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save current settings: %w", err)
	}

	return nil
}

// GetProfiles returns saved overclocking profiles
func (c *WindowsController) GetProfiles(ctx context.Context) ([]*Profile, error) {
	var profiles []*Profile

	files, err := ioutil.ReadDir(c.profilesDir)
	if err != nil {
		return profiles, nil // Return empty list if directory doesn't exist
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" && file.Name() != "_current.json" {
			profilePath := filepath.Join(c.profilesDir, file.Name())
			data, err := ioutil.ReadFile(profilePath)
			if err != nil {
				continue
			}

			var profile Profile
			if err := json.Unmarshal(data, &profile); err != nil {
				continue
			}

			profiles = append(profiles, &profile)
		}
	}

	return profiles, nil
}

// SaveProfile saves an overclocking profile
func (c *WindowsController) SaveProfile(ctx context.Context, profile *Profile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if profile.Name == "_current" {
		return fmt.Errorf("profile name '_current' is reserved")
	}

	// Validate profile settings
	if err := c.validateSettings(profile.Settings); err != nil {
		return fmt.Errorf("invalid profile settings: %w", err)
	}

	profilePath := filepath.Join(c.profilesDir, profile.Name+".json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	if err := ioutil.WriteFile(profilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	return nil
}

// LoadProfile loads an overclocking profile
func (c *WindowsController) LoadProfile(ctx context.Context, profileName string) error {
	if profileName == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	profilePath := filepath.Join(c.profilesDir, profileName+".json")
	data, err := ioutil.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("failed to read profile '%s': %w", profileName, err)
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return fmt.Errorf("failed to parse profile '%s': %w", profileName, err)
	}

	// Apply the profile settings
	return c.SetSettings(ctx, profile.Settings)
}

// detectGPUVendor detects the vendor of the specified GPU
func (c *WindowsController) detectGPUVendor(ctx context.Context, deviceID int) (string, error) {
	// Try nvidia-smi first
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name", "--format=csv,noheader", fmt.Sprintf("--id=%d", deviceID))
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return "nvidia", nil
	}

	// Check for AMD GPUs using WMI or registry
	// For simplicity, we'll try to detect AMD by checking for common AMD GPU names
	cmd = exec.CommandContext(ctx, "wmic", "path", "win32_VideoController", "get", "name", "/format:csv")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect GPU vendor")
	}

	lines := strings.Split(string(output), "\n")
	if deviceID < len(lines)-2 && deviceID >= 0 {
		gpuName := strings.ToLower(strings.TrimSpace(lines[deviceID+2]))
		if strings.Contains(gpuName, "amd") || strings.Contains(gpuName, "radeon") {
			return "amd", nil
		}
	}

	return "unknown", fmt.Errorf("unsupported GPU vendor for device %d", deviceID)
}

// getNvidiaSettings gets current NVIDIA GPU settings using nvidia-smi
func (c *WindowsController) getNvidiaSettings(ctx context.Context, deviceID int) (*Settings, error) {
	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=clocks.gr,clocks.mem,power.limit,temperature.gpu",
		"--format=csv,noheader,nounits",
		fmt.Sprintf("--id=%d", deviceID))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query NVIDIA GPU settings: %w", err)
	}

	fields := strings.Split(strings.TrimSpace(string(output)), ", ")
	if len(fields) < 4 {
		return nil, fmt.Errorf("unexpected nvidia-smi output format")
	}

	settings := &Settings{DeviceID: deviceID}

	// Parse current clocks (these would be base clocks, offsets would need additional queries)
	// Parse core clock
	if _, err := strconv.Atoi(strings.TrimSpace(fields[0])); err == nil {
		// Note: nvidia-smi returns current clocks, not offsets
		// For now, we'll return 0 offsets as we can't easily get base clocks
		settings.CoreClockOffset = 0
	}

	if _, err := strconv.Atoi(strings.TrimSpace(fields[1])); err == nil {
		settings.MemoryClockOffset = 0
	}

	// Parse power limit (would need to compare with default to get percentage)
	if powerLimit, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
		// Assuming default power limit, this is simplified
		settings.PowerLimit = int(powerLimit)
	}

	// Parse temperature limit (simplified)
	if tempLimit, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		settings.TempLimit = int(tempLimit)
	}

	return settings, nil
}

// setNvidiaSettings sets NVIDIA GPU settings using nvidia-smi
func (c *WindowsController) setNvidiaSettings(ctx context.Context, settings *Settings) error {
	// NVIDIA overclocking through nvidia-smi is limited
	// Most overclocking requires nvidia-settings (Linux) or MSI Afterburner/EVGA Precision on Windows
	// nvidia-smi mainly supports power limit changes

	var errors []string

	// Try to set power limit only if specified and not 0
	if settings.PowerLimit > 0 {
		cmd := exec.CommandContext(ctx, "nvidia-smi",
			"-i", fmt.Sprintf("%d", settings.DeviceID),
			"-pl", fmt.Sprintf("%d", settings.PowerLimit))

		if err := cmd.Run(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to set NVIDIA power limit: %v", err))
		}
	}

	// Note: Clock offsets, voltage, and fan speed typically require additional tools
	// like MSI Afterburner or manufacturer-specific utilities on Windows
	if settings.CoreClockOffset != 0 || settings.MemoryClockOffset != 0 {
		errors = append(errors, "clock offset adjustment requires additional tools (MSI Afterburner, EVGA Precision, etc.)")
	}

	if settings.VoltageOffset != 0 {
		errors = append(errors, "voltage offset adjustment requires additional tools (MSI Afterburner, EVGA Precision, etc.)")
	}

	if settings.FanSpeed > 0 {
		errors = append(errors, "fan speed control requires additional tools (MSI Afterburner, EVGA Precision, etc.)")
	}

	// If we have any errors and no power limit was successfully set, return error
	if len(errors) > 0 {
		if settings.PowerLimit > 0 {
			// Power limit was attempted but failed
			return fmt.Errorf(strings.Join(errors, "; "))
		} else {
			// Only unsupported operations were requested
			return fmt.Errorf(strings.Join(errors, "; "))
		}
	}

	return nil
}

// getAMDSettings gets current AMD GPU settings using OverdriveNTool
func (c *WindowsController) getAMDSettings(ctx context.Context, deviceID int) (*Settings, error) {
	// Check if OverdriveNTool is available
	cmd := exec.CommandContext(ctx, "OverdriveNTool.exe", "-r", fmt.Sprintf("%d", deviceID))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("OverdriveNTool not available or failed: %w", err)
	}

	settings := &Settings{DeviceID: deviceID}

	// Parse OverdriveNTool output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse GPU clock
		if strings.Contains(line, "GPU_P7") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if _, err := strconv.Atoi(parts[2]); err == nil {
					// This is current clock, we'd need base clock to calculate offset
					settings.CoreClockOffset = 0 // Simplified
				}
			}
		}

		// Parse memory clock
		if strings.Contains(line, "Mem_P3") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if _, err := strconv.Atoi(parts[2]); err == nil {
					settings.MemoryClockOffset = 0 // Simplified
				}
			}
		}

		// Parse power limit
		if strings.Contains(line, "Power_Limit") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if power, err := strconv.Atoi(parts[1]); err == nil {
					settings.PowerLimit = power
				}
			}
		}

		// Parse temperature limit
		if strings.Contains(line, "Temp_Limit") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if temp, err := strconv.Atoi(parts[1]); err == nil {
					settings.TempLimit = temp
				}
			}
		}

		// Parse fan speed
		if strings.Contains(line, "Fan_Min") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if fan, err := strconv.Atoi(parts[1]); err == nil {
					settings.FanSpeed = fan
				}
			}
		}
	}

	return settings, nil
}

// setAMDSettings sets AMD GPU settings using OverdriveNTool
func (c *WindowsController) setAMDSettings(ctx context.Context, settings *Settings) error {
	// Create temporary config file for OverdriveNTool
	configPath := filepath.Join(c.profilesDir, fmt.Sprintf("temp_config_%d.txt", settings.DeviceID))

	var configLines []string

	// Add GPU clock offset
	if settings.CoreClockOffset != 0 {
		configLines = append(configLines, fmt.Sprintf("GPU_P7: %d", settings.CoreClockOffset))
	}

	// Add memory clock offset
	if settings.MemoryClockOffset != 0 {
		configLines = append(configLines, fmt.Sprintf("Mem_P3: %d", settings.MemoryClockOffset))
	}

	// Add power limit
	if settings.PowerLimit > 0 {
		configLines = append(configLines, fmt.Sprintf("Power_Limit: %d", settings.PowerLimit))
	}

	// Add temperature limit
	if settings.TempLimit > 0 {
		configLines = append(configLines, fmt.Sprintf("Temp_Limit: %d", settings.TempLimit))
	}

	// Add fan speed
	if settings.FanSpeed > 0 {
		configLines = append(configLines, fmt.Sprintf("Fan_Min: %d", settings.FanSpeed))
		configLines = append(configLines, fmt.Sprintf("Fan_Max: %d", settings.FanSpeed))
	}

	// Add voltage offset if specified
	if settings.VoltageOffset != 0 {
		configLines = append(configLines, fmt.Sprintf("Voltage_GPU: %.0f", settings.VoltageOffset))
	}

	// Write config file
	configContent := strings.Join(configLines, "\n")
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create OverdriveNTool config: %w", err)
	}

	// Apply settings using OverdriveNTool
	cmd := exec.CommandContext(ctx, "OverdriveNTool.exe",
		"-p", fmt.Sprintf("%d", settings.DeviceID),
		"-f", configPath)

	if err := cmd.Run(); err != nil {
		// Clean up config file
		os.Remove(configPath)
		return fmt.Errorf("failed to apply AMD settings with OverdriveNTool: %w", err)
	}

	// Clean up config file
	os.Remove(configPath)

	return nil
}

// validateSettings validates overclocking settings for safety
func (c *WindowsController) validateSettings(settings *Settings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	// Validate core clock offset (reasonable limits) - only if set
	if settings.CoreClockOffset != 0 && (settings.CoreClockOffset < -500 || settings.CoreClockOffset > 500) {
		return fmt.Errorf("core clock offset must be between -500 and +500 MHz")
	}

	// Validate memory clock offset - only if set
	if settings.MemoryClockOffset != 0 && (settings.MemoryClockOffset < -1000 || settings.MemoryClockOffset > 1000) {
		return fmt.Errorf("memory clock offset must be between -1000 and +1000 MHz")
	}

	// Validate power limit - only if set
	if settings.PowerLimit != 0 && (settings.PowerLimit < 50 || settings.PowerLimit > 150) {
		return fmt.Errorf("power limit must be between 50%% and 150%%")
	}

	// Validate temperature limit - only if set
	if settings.TempLimit != 0 && (settings.TempLimit < 60 || settings.TempLimit > 95) {
		return fmt.Errorf("temperature limit must be between 60°C and 95°C")
	}

	// Validate fan speed - only if set (0 is auto, so check for negative or > 100)
	if settings.FanSpeed < 0 || settings.FanSpeed > 100 {
		return fmt.Errorf("fan speed must be between 0%% and 100%%")
	}

	// Validate voltage offset - only if set
	if settings.VoltageOffset != 0 && (settings.VoltageOffset < -100 || settings.VoltageOffset > 100) {
		return fmt.Errorf("voltage offset must be between -100mV and +100mV")
	}

	return nil
}
