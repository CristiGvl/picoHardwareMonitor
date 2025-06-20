//go:build linux

package temps

import (
	"context"

	"github.com/shirou/gopsutil/v3/host"
)

// LinuxReader implements temperature monitoring for Linux
type LinuxReader struct{}

// newPlatformReader creates a new Linux temperature reader
func newPlatformReader() Reader {
	return &LinuxReader{}
}

// GetInfo returns temperature information
func (r *LinuxReader) GetInfo(ctx context.Context) (*Info, error) {
	temps, err := host.SensorsTemperaturesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	info := &Info{
		CPU:    []*Sensor{},
		GPU:    []*Sensor{},
		System: []*Sensor{},
		Drives: []*Sensor{},
	}

	for _, temp := range temps {
		sensor := &Sensor{
			Name:        temp.SensorKey,
			Label:       temp.SensorKey,
			Temperature: temp.Temperature,
			Critical:    temp.Critical,
			Max:         temp.High,
		}

		// Categorize sensors based on their names
		switch {
		case containsAny(temp.SensorKey, []string{"cpu", "core", "processor"}):
			info.CPU = append(info.CPU, sensor)
		case containsAny(temp.SensorKey, []string{"gpu", "nvidia", "amd", "radeon"}):
			info.GPU = append(info.GPU, sensor)
		case containsAny(temp.SensorKey, []string{"drive", "disk", "nvme", "sda", "sdb"}):
			info.Drives = append(info.Drives, sensor)
		default:
			info.System = append(info.System, sensor)
		}
	}

	return info, nil
}

func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
