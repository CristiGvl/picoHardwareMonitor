//go:build windows

package memory

import (
	"context"

	"github.com/shirou/gopsutil/v3/mem"
)

// WindowsReader implements memory monitoring for Windows
type WindowsReader struct{}

// newPlatformReader creates a new Windows memory reader
func newPlatformReader() Reader {
	return &WindowsReader{}
}

// GetInfo returns memory information
func (r *WindowsReader) GetInfo(ctx context.Context) (*Info, error) {
	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}

	info := &Info{
		Total:     memInfo.Total / (1024 * 1024),     // Convert to MB
		Used:      memInfo.Used / (1024 * 1024),      // Convert to MB
		Available: memInfo.Available / (1024 * 1024), // Convert to MB
		Usage:     memInfo.UsedPercent,
	}

	return info, nil
}
