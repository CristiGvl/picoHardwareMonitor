//go:build windows

package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/CristiGvl/picoHWMon/internal/overclock"
	"github.com/StackExchange/wmi"
)

// WindowsReader implements GPU monitoring for Windows
type WindowsReader struct {
	overclockController overclock.Controller
}

// newPlatformReader creates a new Windows GPU reader
func newPlatformReader() Reader {
	return &WindowsReader{
		overclockController: overclock.NewController(),
	}
}

// Win32_VideoController represents WMI video controller
type Win32_VideoController struct {
	Name                        string
	AdapterRAM                  uint32
	VideoProcessor              string
	DriverVersion               string
	CurrentHorizontalResolution uint32
	CurrentVerticalResolution   uint32
}

// GetInfo returns GPU information
func (r *WindowsReader) GetInfo(ctx context.Context) ([]*Info, error) {
	var gpus []*Info

	// Try NVIDIA GPUs first
	nvidiaGPUs, err := r.getNvidiaGPUs(ctx)
	if err == nil {
		gpus = append(gpus, nvidiaGPUs...)
	}

	// Fallback to WMI for basic info
	wmiGPUs, err := r.getWMIGPUs(ctx)
	if err == nil {
		// Filter out duplicates if NVIDIA was already detected
		for _, wmiGPU := range wmiGPUs {
			isDuplicate := false
			for _, existing := range gpus {
				if strings.Contains(strings.ToLower(wmiGPU.Model), strings.ToLower(existing.Model)) {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				gpus = append(gpus, wmiGPU)
			}
		}
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs found")
	}

	return gpus, nil
}

func (r *WindowsReader) getNvidiaGPUs(ctx context.Context) ([]*Info, error) {
	// Check if nvidia-smi is available
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name,memory.total,memory.used,utilization.gpu,utilization.memory,temperature.gpu,power.draw,clocks.gr,clocks.mem", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi not available: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var gpus []*Info

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 9 {
			continue
		}

		info := &Info{
			Vendor: NVIDIA,
			Model:  strings.TrimSpace(fields[0]),
		}

		// Parse VRAM total
		if vramTotal, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64); err == nil {
			info.VRAM = vramTotal
		}

		// Parse GPU utilization
		if usage, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
			info.Usage = usage
		}

		// Parse memory utilization
		if memUsage, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64); err == nil {
			info.MemoryUsage = memUsage
		}

		// Parse temperature
		if temp, err := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64); err == nil {
			info.Temperature = temp
		}

		// Parse power usage
		if power, err := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64); err == nil {
			info.PowerUsage = power
		}

		// Parse core clock
		if coreClock, err := strconv.Atoi(strings.TrimSpace(fields[7])); err == nil {
			info.ClockCore = coreClock
		}

		// Parse memory clock
		if memClock, err := strconv.Atoi(strings.TrimSpace(fields[8])); err == nil {
			info.ClockMemory = memClock
		}

		gpus = append(gpus, info)
	}

	return gpus, nil
}

func (r *WindowsReader) getWMIGPUs(ctx context.Context) ([]*Info, error) {
	var videoControllers []Win32_VideoController
	err := wmi.Query("SELECT Name, AdapterRAM, VideoProcessor FROM Win32_VideoController", &videoControllers)
	if err != nil {
		return nil, fmt.Errorf("failed to query WMI: %w", err)
	}

	var gpus []*Info
	for _, controller := range videoControllers {
		if controller.Name == "" {
			continue
		}

		vendor := Unknown
		lowerName := strings.ToLower(controller.Name)
		if strings.Contains(lowerName, "nvidia") || strings.Contains(lowerName, "geforce") || strings.Contains(lowerName, "quadro") {
			vendor = NVIDIA
		} else if strings.Contains(lowerName, "amd") || strings.Contains(lowerName, "radeon") {
			vendor = AMD
		} else if strings.Contains(lowerName, "intel") {
			vendor = Intel
		}

		info := &Info{
			Vendor: vendor,
			Model:  controller.Name,
			VRAM:   uint64(controller.AdapterRAM / (1024 * 1024)), // Convert bytes to MB
		}

		gpus = append(gpus, info)
	}

	return gpus, nil
}

// GetOverclockSettings returns current overclock settings
func (r *WindowsReader) GetOverclockSettings(ctx context.Context, deviceID int) (*OverclockSettings, error) {
	settings, err := r.overclockController.GetSettings(ctx, deviceID)
	if err != nil {
		return &OverclockSettings{}, err
	}

	return &OverclockSettings{
		CoreClockOffset:   settings.CoreClockOffset,
		MemoryClockOffset: settings.MemoryClockOffset,
		PowerLimit:        settings.PowerLimit,
		FanSpeed:          settings.FanSpeed,
	}, nil
}

// SetOverclockSettings applies overclock settings
func (r *WindowsReader) SetOverclockSettings(ctx context.Context, deviceID int, settings *OverclockSettings) (*OverclockResult, error) {
	overclockSettings := &overclock.Settings{
		DeviceID:          deviceID,
		CoreClockOffset:   settings.CoreClockOffset,
		MemoryClockOffset: settings.MemoryClockOffset,
		PowerLimit:        settings.PowerLimit,
		TempLimit:         83, // Default temp limit
		FanSpeed:          settings.FanSpeed,
		VoltageOffset:     0, // Default voltage offset
	}

	err := r.overclockController.SetSettings(ctx, overclockSettings)
	if err != nil {
		return &OverclockResult{
			Success:  false,
			Applied:  []string{},
			Warnings: []string{},
			Errors:   []string{err.Error()},
		}, nil
	}

	// Build list of applied settings
	var applied []string
	var warnings []string

	if settings.CoreClockOffset != 0 {
		applied = append(applied, fmt.Sprintf("Core clock offset: %+d MHz", settings.CoreClockOffset))
	}
	if settings.MemoryClockOffset != 0 {
		applied = append(applied, fmt.Sprintf("Memory clock offset: %+d MHz", settings.MemoryClockOffset))
	}
	if settings.PowerLimit > 0 {
		applied = append(applied, fmt.Sprintf("Power limit: %d%%", settings.PowerLimit))
	}
	// Temperature limit is handled internally with default value
	if settings.FanSpeed > 0 {
		applied = append(applied, fmt.Sprintf("Fan speed: %d%%", settings.FanSpeed))
	}
	// Voltage offset is handled internally with default value

	// Add warnings for NVIDIA limitations
	vendor, _ := r.detectGPUVendor(ctx, deviceID)
	if vendor == "nvidia" {
		if settings.CoreClockOffset != 0 || settings.MemoryClockOffset != 0 || settings.FanSpeed > 0 {
			warnings = append(warnings, "NVIDIA clock/voltage/fan control requires additional tools (MSI Afterburner, EVGA Precision, etc.)")
		}
	}

	return &OverclockResult{
		Success:  true,
		Applied:  applied,
		Warnings: warnings,
		Errors:   []string{},
	}, nil
}

// detectGPUVendor detects the vendor of the specified GPU
func (r *WindowsReader) detectGPUVendor(ctx context.Context, deviceID int) (string, error) {
	// Try nvidia-smi first
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name", "--format=csv,noheader", fmt.Sprintf("--id=%d", deviceID))
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return "nvidia", nil
	}

	// Check for AMD GPUs using WMI
	var videoControllers []Win32_VideoController
	err = wmi.Query("SELECT Name FROM Win32_VideoController", &videoControllers)
	if err != nil {
		return "", fmt.Errorf("failed to detect GPU vendor")
	}

	if deviceID < len(videoControllers) && deviceID >= 0 {
		gpuName := strings.ToLower(videoControllers[deviceID].Name)
		if strings.Contains(gpuName, "amd") || strings.Contains(gpuName, "radeon") {
			return "amd", nil
		}
		if strings.Contains(gpuName, "nvidia") || strings.Contains(gpuName, "geforce") || strings.Contains(gpuName, "quadro") {
			return "nvidia", nil
		}
	}

	return "unknown", fmt.Errorf("unsupported GPU vendor for device %d", deviceID)
}
