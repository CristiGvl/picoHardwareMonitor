//go:build !linux && !windows

package overclock

import (
	"context"
	"fmt"
)

// UnsupportedController is a fallback for unsupported platforms
type UnsupportedController struct{}

// newPlatformController creates a fallback overclocking controller for unsupported platforms
func newPlatformController() Controller {
	return &UnsupportedController{}
}

// GetSettings returns an error for unsupported platforms
func (c *UnsupportedController) GetSettings(ctx context.Context, deviceID int) (*Settings, error) {
	return nil, fmt.Errorf("overclocking control not supported on this platform")
}

// SetSettings returns an error for unsupported platforms
func (c *UnsupportedController) SetSettings(ctx context.Context, settings *Settings) error {
	return fmt.Errorf("overclocking control not supported on this platform")
}

// GetProfiles returns an error for unsupported platforms
func (c *UnsupportedController) GetProfiles(ctx context.Context) ([]*Profile, error) {
	return nil, fmt.Errorf("overclocking control not supported on this platform")
}

// SaveProfile returns an error for unsupported platforms
func (c *UnsupportedController) SaveProfile(ctx context.Context, profile *Profile) error {
	return fmt.Errorf("overclocking control not supported on this platform")
}

// LoadProfile returns an error for unsupported platforms
func (c *UnsupportedController) LoadProfile(ctx context.Context, profileName string) error {
	return fmt.Errorf("overclocking control not supported on this platform")
}
