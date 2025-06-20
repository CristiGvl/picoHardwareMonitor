//go:build !linux && !windows

package fan

import (
	"context"
	"fmt"
)

// UnsupportedController is a fallback for unsupported platforms
type UnsupportedController struct{}

// newPlatformController creates a fallback fan controller for unsupported platforms
func newPlatformController() Controller {
	return &UnsupportedController{}
}

// GetFans returns an error for unsupported platforms
func (c *UnsupportedController) GetFans(ctx context.Context) ([]*Info, error) {
	return nil, fmt.Errorf("fan control not supported on this platform")
}

// GetSettings returns an error for unsupported platforms
func (c *UnsupportedController) GetSettings(ctx context.Context, fanID int) (*Settings, error) {
	return nil, fmt.Errorf("fan control not supported on this platform")
}

// SetSettings returns an error for unsupported platforms
func (c *UnsupportedController) SetSettings(ctx context.Context, fanID int, settings *Settings) error {
	return fmt.Errorf("fan control not supported on this platform")
}
