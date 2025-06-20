//go:build !linux && !windows

package cpu

import (
	"context"
	"fmt"
)

// UnsupportedReader is a fallback for unsupported platforms
type UnsupportedReader struct{}

// newPlatformReader creates a fallback CPU reader for unsupported platforms
func newPlatformReader() Reader {
	return &UnsupportedReader{}
}

// GetInfo returns an error for unsupported platforms
func (r *UnsupportedReader) GetInfo(ctx context.Context) (*Info, error) {
	return nil, fmt.Errorf("CPU monitoring not supported on this platform")
}

// GetUsage returns an error for unsupported platforms
func (r *UnsupportedReader) GetUsage(ctx context.Context) (float64, error) {
	return 0, fmt.Errorf("CPU monitoring not supported on this platform")
}
