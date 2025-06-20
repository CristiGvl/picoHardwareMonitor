//go:build linux

package cpu

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// LinuxReader implements CPU monitoring for Linux
type LinuxReader struct{}

// newPlatformReader creates a new Linux CPU reader
func newPlatformReader() Reader {
	return &LinuxReader{}
}

// GetInfo returns CPU information
func (r *LinuxReader) GetInfo(ctx context.Context) (*Info, error) {
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

	// Calculate actual physical cores and logical threads
	physicalCores := int(cpuInfo[0].Cores)
	logicalThreads := len(cpuInfo)

	// Always try to get accurate core count from /proc/cpuinfo
	if coreCount := r.getPhysicalCoreCount(); coreCount > 0 {
		physicalCores = coreCount
	} else if physicalCores == 0 || physicalCores > logicalThreads {
		// Fallback: For Intel i7-3770 (4 cores, 8 threads), assume cores = threads/2 if hyperthreading
		if logicalThreads > 4 && (logicalThreads%2 == 0) {
			physicalCores = logicalThreads / 2
		} else {
			physicalCores = logicalThreads
		}
	}

	info := &Info{
		Model:     cpuInfo[0].ModelName,
		Cores:     physicalCores,
		Threads:   logicalThreads,
		Usage:     usage,
		Frequency: cpuInfo[0].Mhz,
	}

	return info, nil
}

// GetUsage returns CPU usage percentage
func (r *LinuxReader) GetUsage(ctx context.Context) (float64, error) {
	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, nil
	}

	return percentages[0], nil
}

// getPhysicalCoreCount reads /proc/cpuinfo to get accurate physical core count
func (r *LinuxReader) getPhysicalCoreCount() int {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(content), "\n")
	coreMap := make(map[string]bool)
	coreCountFromHeader := 0

	var currentPhysicalID, currentCoreID string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// First try to get cores from "cpu cores" field
		if strings.HasPrefix(line, "cpu cores") && coreCountFromHeader == 0 {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if cores, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					coreCountFromHeader = cores
				}
			}
		}

		if strings.HasPrefix(line, "physical id") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentPhysicalID = strings.TrimSpace(parts[1])
			}
		}

		if strings.HasPrefix(line, "core id") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentCoreID = strings.TrimSpace(parts[1])
			}
		}

		// When we hit an empty line, we've finished processing one logical CPU
		if line == "" && currentPhysicalID != "" && currentCoreID != "" {
			coreKey := currentPhysicalID + ":" + currentCoreID
			coreMap[coreKey] = true
			currentPhysicalID = ""
			currentCoreID = ""
		}
	}

	// Process the last entry if file doesn't end with empty line
	if currentPhysicalID != "" && currentCoreID != "" {
		coreKey := currentPhysicalID + ":" + currentCoreID
		coreMap[coreKey] = true
	}

	// Prefer the count from coreMap, but fallback to header value if available
	if len(coreMap) > 0 {
		return len(coreMap)
	}
	return coreCountFromHeader
}
