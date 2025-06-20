//go:build windows

package disk

import (
	"context"

	"github.com/shirou/gopsutil/v3/disk"
)

// WindowsReader implements disk monitoring for Windows
type WindowsReader struct{}

// newPlatformReader creates a new Windows disk reader
func newPlatformReader() Reader {
	return &WindowsReader{}
}

// GetInfo returns disk information
func (r *WindowsReader) GetInfo(ctx context.Context) ([]*Info, error) {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, err
	}

	var disks []*Info
	for _, partition := range partitions {
		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			continue // Skip partitions we can't read
		}

		info := &Info{
			Device:     partition.Device,
			Mountpoint: partition.Mountpoint,
			Filesystem: partition.Fstype,
			Total:      usage.Total / (1024 * 1024), // Convert to MB
			Used:       usage.Used / (1024 * 1024),  // Convert to MB
			Available:  usage.Free / (1024 * 1024),  // Convert to MB
			Usage:      usage.UsedPercent,
		}

		disks = append(disks, info)
	}

	return disks, nil
}
