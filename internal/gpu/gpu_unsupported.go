//go:build !linux && !windows

package gpu

import (
	"context"
	"fmt"
)

// UnsupportedReader is a fallback for unsupported platforms
type UnsupportedReader struct{}

// newPlatformReader creates a fallback GPU reader for unsupported platforms
func newPlatformReader() Reader {
	return &UnsupportedReader{}
}

// GetInfo returns an error for unsupported platforms
func (r *UnsupportedReader) GetInfo(ctx context.Context) ([]*Info, error) {
	return nil, fmt.Errorf("GPU monitoring not supported on this platform")
}

// GetOverclockSettings returns an error for unsupported platforms
func (r *UnsupportedReader) GetOverclockSettings(ctx context.Context, deviceID int) (*OverclockSettings, error) {
	return nil, fmt.Errorf("GPU overclocking not supported on this platform")
}

// SetOverclockSettings returns an error for unsupported platforms
func (r *UnsupportedReader) SetOverclockSettings(ctx context.Context, deviceID int, settings *OverclockSettings) (*OverclockResult, error) {
	return nil, fmt.Errorf("GPU overclocking not supported on this platform")
}
