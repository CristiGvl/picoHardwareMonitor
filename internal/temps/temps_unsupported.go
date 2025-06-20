//go:build !linux && !windows

package temps

import (
	"context"
	"fmt"
)

// UnsupportedReader is a fallback for unsupported platforms
type UnsupportedReader struct{}

// newPlatformReader creates a fallback temperature reader for unsupported platforms
func newPlatformReader() Reader {
	return &UnsupportedReader{}
}

// GetInfo returns an error for unsupported platforms
func (r *UnsupportedReader) GetInfo(ctx context.Context) (*Info, error) {
	return nil, fmt.Errorf("temperature monitoring not supported on this platform")
}
