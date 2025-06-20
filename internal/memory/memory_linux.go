//go:build linux

package memory

import (
	"context"

	"github.com/shirou/gopsutil/v3/mem"
)

// LinuxReader implements memory monitoring for Linux
type LinuxReader struct{}

// newPlatformReader creates a new Linux memory reader
func newPlatformReader() Reader {
	return &LinuxReader{}
}

// GetInfo returns memory information
func (r *LinuxReader) GetInfo(ctx context.Context) (*Info, error) {
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
