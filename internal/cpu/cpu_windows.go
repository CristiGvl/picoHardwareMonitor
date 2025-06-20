//go:build windows

package cpu

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// WindowsReader implements CPU monitoring for Windows
type WindowsReader struct{}

// newPlatformReader creates a new Windows CPU reader
func newPlatformReader() Reader {
	return &WindowsReader{}
}

// GetInfo returns CPU information
func (r *WindowsReader) GetInfo(ctx context.Context) (*Info, error) {
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(cpuInfo) == 0 {
		return nil, nil
	}

	usage, err := r.GetUsage(ctx)
	if err != nil {
		usage = 0 // fallback to 0 if we can't get usage
	}

	info := &Info{
		Model:     cpuInfo[0].ModelName,
		Cores:     int(cpuInfo[0].Cores),
		Threads:   len(cpuInfo),
		Usage:     usage,
		Frequency: cpuInfo[0].Mhz,
	}

	return info, nil
}

// GetUsage returns CPU usage percentage
func (r *WindowsReader) GetUsage(ctx context.Context) (float64, error) {
	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, nil
	}

	return percentages[0], nil
}
