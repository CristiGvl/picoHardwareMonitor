//go:build linux

package overclock

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// LinuxController implements overclocking control for Linux
type LinuxController struct {
	profilesDir string
}

// newPlatformController creates a new Linux overclocking controller
func newPlatformController() Controller {
	homeDir, _ := os.UserHomeDir()
	profilesDir := filepath.Join(homeDir, ".config", "picohwmon", "profiles")

	// Create profiles directory if it doesn't exist
	os.MkdirAll(profilesDir, 0755)

	return &LinuxController{
		profilesDir: profilesDir,
	}
}

// GetSettings returns current overclocking settings
func (c *LinuxController) GetSettings(ctx context.Context, deviceID int) (*Settings, error) {
	// This would typically read from hardware or configuration
	// For now, return default settings
	return &Settings{
		DeviceID:          deviceID,
		CoreClockOffset:   0,
		MemoryClockOffset: 0,
		PowerLimit:        100,
		TempLimit:         83,
		FanSpeed:          0, // Auto
		VoltageOffset:     0,
	}, nil
}

// SetSettings applies overclocking settings
func (c *LinuxController) SetSettings(ctx context.Context, settings *Settings) error {
	// Validate settings
	if err := c.validateSettings(settings); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
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
func (c *LinuxController) GetProfiles(ctx context.Context) ([]*Profile, error) {
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
func (c *LinuxController) SaveProfile(ctx context.Context, profile *Profile) error {
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
func (c *LinuxController) LoadProfile(ctx context.Context, profileName string) error {
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

// validateSettings validates overclocking settings for safety
func (c *LinuxController) validateSettings(settings *Settings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	// Validate core clock offset (reasonable limits)
	if settings.CoreClockOffset < -500 || settings.CoreClockOffset > 500 {
		return fmt.Errorf("core clock offset must be between -500 and +500 MHz")
	}

	// Validate memory clock offset
	if settings.MemoryClockOffset < -1000 || settings.MemoryClockOffset > 1000 {
		return fmt.Errorf("memory clock offset must be between -1000 and +1000 MHz")
	}

	// Validate power limit
	if settings.PowerLimit < 50 || settings.PowerLimit > 150 {
		return fmt.Errorf("power limit must be between 50%% and 150%%")
	}

	// Validate temperature limit
	if settings.TempLimit < 60 || settings.TempLimit > 95 {
		return fmt.Errorf("temperature limit must be between 60°C and 95°C")
	}

	// Validate fan speed
	if settings.FanSpeed < 0 || settings.FanSpeed > 100 {
		return fmt.Errorf("fan speed must be between 0%% and 100%%")
	}

	// Validate voltage offset
	if settings.VoltageOffset < -100 || settings.VoltageOffset > 100 {
		return fmt.Errorf("voltage offset must be between -100mV and +100mV")
	}

	return nil
}
