//go:build linux

package disk

import (
	"context"

	"github.com/shirou/gopsutil/v3/disk"
)

// LinuxReader implements disk monitoring for Linux
type LinuxReader struct{}

// newPlatformReader creates a new Linux disk reader
func newPlatformReader() Reader {
	return &LinuxReader{}
}

// GetInfo returns disk information
func (r *LinuxReader) GetInfo(ctx context.Context) ([]*Info, error) {
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
