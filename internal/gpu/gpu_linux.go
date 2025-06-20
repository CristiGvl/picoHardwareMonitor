//go:build linux

package gpu

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// LinuxReader implements GPU monitoring for Linux
type LinuxReader struct{}

// newPlatformReader creates a new Linux GPU reader
func newPlatformReader() Reader {
	return &LinuxReader{}
}

// NvidiaSMIOutput represents nvidia-smi XML output structure
type NvidiaSMIOutput struct {
	GPUs []struct {
		ProductName string `xml:"product_name"`
		MemoryInfo  struct {
			Total string `xml:"total"`
			Used  string `xml:"used"`
		} `xml:"fb_memory_usage"`
		Utilization struct {
			GPU    string `xml:"gpu_util"`
			Memory string `xml:"memory_util"`
		} `xml:"utilization"`
		Temperature struct {
			Current string `xml:"gpu_temp"`
		} `xml:"temperature"`
		PowerReadings struct {
			PowerDraw string `xml:"power_draw"`
		} `xml:"power_readings"`
		ClocksSM struct {
			GraphicsClock string `xml:"graphics_clock"`
			MemoryClock   string `xml:"mem_clock"`
		} `xml:"clocks"`
	} `xml:"gpu"`
}

// GetInfo returns GPU information
func (r *LinuxReader) GetInfo(ctx context.Context) ([]*Info, error) {
	var gpus []*Info

	// Try NVIDIA GPUs first
	nvidiaGPUs, err := r.getNvidiaGPUs(ctx)
	if err == nil {
		gpus = append(gpus, nvidiaGPUs...)
	}

	// Try AMD GPUs
	amdGPUs, err := r.getAMDGPUs(ctx)
	if err == nil {
		gpus = append(gpus, amdGPUs...)
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("no supported GPUs found")
	}

	return gpus, nil
}

func (r *LinuxReader) getNvidiaGPUs(ctx context.Context) ([]*Info, error) {
	cmd := exec.CommandContext(ctx, "nvidia-smi", "-q", "-x")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi not available: %w", err)
	}

	var smiOutput NvidiaSMIOutput
	if err := xml.Unmarshal(output, &smiOutput); err != nil {
		return nil, fmt.Errorf("failed to parse nvidia-smi output: %w", err)
	}

	var gpus []*Info
	for _, gpu := range smiOutput.GPUs {
		info := &Info{
			Vendor: NVIDIA,
			Model:  gpu.ProductName,
		}

		// Parse VRAM
		if vramStr := strings.TrimSuffix(gpu.MemoryInfo.Total, " MiB"); vramStr != "" {
			if vram, err := strconv.ParseUint(vramStr, 10, 64); err == nil {
				info.VRAM = vram
			}
		}

		// Parse GPU usage
		if usageStr := strings.TrimSuffix(gpu.Utilization.GPU, " %"); usageStr != "" {
			if usage, err := strconv.ParseFloat(usageStr, 64); err == nil {
				info.Usage = usage
			}
		}

		// Parse memory usage
		if memUsageStr := strings.TrimSuffix(gpu.Utilization.Memory, " %"); memUsageStr != "" {
			if memUsage, err := strconv.ParseFloat(memUsageStr, 64); err == nil {
				info.MemoryUsage = memUsage
			}
		}

		// Parse temperature
		if tempStr := strings.TrimSuffix(gpu.Temperature.Current, " C"); tempStr != "" {
			if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
				info.Temperature = temp
			}
		}

		// Parse power usage
		if powerStr := strings.TrimSuffix(gpu.PowerReadings.PowerDraw, " W"); powerStr != "" {
			if power, err := strconv.ParseFloat(powerStr, 64); err == nil {
				info.PowerUsage = power
			}
		}

		// Parse clocks
		if coreClockStr := strings.TrimSuffix(gpu.ClocksSM.GraphicsClock, " MHz"); coreClockStr != "" {
			if coreClock, err := strconv.Atoi(coreClockStr); err == nil {
				info.ClockCore = coreClock
			}
		}

		if memClockStr := strings.TrimSuffix(gpu.ClocksSM.MemoryClock, " MHz"); memClockStr != "" {
			if memClock, err := strconv.Atoi(memClockStr); err == nil {
				info.ClockMemory = memClock
			}
		}

		gpus = append(gpus, info)
	}

	return gpus, nil
}

func (r *LinuxReader) getAMDGPUs(ctx context.Context) ([]*Info, error) {
	// Try rocm-smi first
	cmd := exec.CommandContext(ctx, "rocm-smi", "--showproductname", "--showuse", "--showtemp", "--showmemuse")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to sysfs
		return r.getAMDGPUsFromSysfs(ctx)
	}

	lines := strings.Split(string(output), "\n")
	var gpus []*Info

	for _, line := range lines {
		if strings.Contains(line, "card") && !strings.Contains(line, "=") {
			// Parse rocm-smi output - this is a simplified parser
			info := &Info{
				Vendor: AMD,
				Model:  "AMD GPU", // rocm-smi output parsing would need more detailed implementation
			}
			gpus = append(gpus, info)
		}
	}

	return gpus, nil
}

func (r *LinuxReader) getAMDGPUsFromSysfs(ctx context.Context) ([]*Info, error) {
	var gpus []*Info

	// Look for AMD GPUs in /sys/class/drm/
	drmPath := "/sys/class/drm"
	entries, err := os.ReadDir(drmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DRM directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "card") || strings.Contains(entry.Name(), "-") {
			continue
		}

		cardPath := filepath.Join(drmPath, entry.Name(), "device")

		// Check if it's an AMD GPU by reading vendor ID
		vendorPath := filepath.Join(cardPath, "vendor")
		vendorData, err := os.ReadFile(vendorPath)
		if err != nil {
			continue
		}

		vendor := strings.TrimSpace(string(vendorData))
		if vendor != "0x1002" { // AMD vendor ID
			continue
		}

		gpu := &Info{
			Vendor: AMD,
			Model:  "AMD GPU",
		}

		// Try to get more specific model name
		if devicePath := filepath.Join(cardPath, "device"); fileExists(devicePath) {
			if deviceData, err := os.ReadFile(devicePath); err == nil {
				deviceID := strings.TrimSpace(string(deviceData))
				gpu.Model = fmt.Sprintf("AMD GPU (Device ID: %s)", deviceID)
			}
		}

		// Try to get GPU usage from GPU busy percentage
		if busyPath := filepath.Join(cardPath, "gpu_busy_percent"); fileExists(busyPath) {
			if busyData, err := os.ReadFile(busyPath); err == nil {
				if usage, err := strconv.ParseFloat(strings.TrimSpace(string(busyData)), 64); err == nil {
					gpu.Usage = usage
				}
			}
		}

		// Try to get VRAM info
		if vramTotalPath := filepath.Join(cardPath, "mem_info_vram_total"); fileExists(vramTotalPath) {
			if vramData, err := os.ReadFile(vramTotalPath); err == nil {
				if vram, err := strconv.ParseUint(strings.TrimSpace(string(vramData)), 10, 64); err == nil {
					gpu.VRAM = vram / (1024 * 1024) // Convert bytes to MB
				}
			}
		}

		// Try to get VRAM usage
		if vramUsedPath := filepath.Join(cardPath, "mem_info_vram_used"); fileExists(vramUsedPath) {
			if vramUsedData, err := os.ReadFile(vramUsedPath); err == nil {
				if vramUsed, err := strconv.ParseUint(strings.TrimSpace(string(vramUsedData)), 10, 64); err == nil && gpu.VRAM > 0 {
					vramUsedMB := vramUsed / (1024 * 1024)
					gpu.MemoryUsage = float64(vramUsedMB) / float64(gpu.VRAM) * 100
				}
			}
		}

		// Try to get temperature from hwmon
		if hwmonPath := findAMDHwmonPath(cardPath); hwmonPath != "" {
			if tempPath := filepath.Join(hwmonPath, "temp1_input"); fileExists(tempPath) {
				if tempData, err := os.ReadFile(tempPath); err == nil {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64); err == nil {
						gpu.Temperature = temp / 1000 // Convert millidegrees to degrees
					}
				}
			}

			// Try to get power usage
			if powerPath := filepath.Join(hwmonPath, "power1_average"); fileExists(powerPath) {
				if powerData, err := os.ReadFile(powerPath); err == nil {
					if power, err := strconv.ParseFloat(strings.TrimSpace(string(powerData)), 64); err == nil {
						gpu.PowerUsage = power / 1000000 // Convert microwatts to watts
					}
				}
			}
		}

		// Try to get GPU clocks
		if freqPath := filepath.Join(cardPath, "pp_dpm_sclk"); fileExists(freqPath) {
			if freqData, err := os.ReadFile(freqPath); err == nil {
				lines := strings.Split(string(freqData), "\n")
				for _, line := range lines {
					if strings.Contains(line, "*") { // Current frequency marked with *
						parts := strings.Fields(line)
						if len(parts) >= 2 {
							freqStr := strings.TrimSuffix(parts[1], "Mhz")
							if freq, err := strconv.Atoi(freqStr); err == nil {
								gpu.ClockCore = freq
							}
						}
						break
					}
				}
			}
		}

		// Try to get memory clocks
		if memFreqPath := filepath.Join(cardPath, "pp_dpm_mclk"); fileExists(memFreqPath) {
			if memFreqData, err := os.ReadFile(memFreqPath); err == nil {
				lines := strings.Split(string(memFreqData), "\n")
				for _, line := range lines {
					if strings.Contains(line, "*") { // Current frequency marked with *
						parts := strings.Fields(line)
						if len(parts) >= 2 {
							freqStr := strings.TrimSuffix(parts[1], "Mhz")
							if freq, err := strconv.Atoi(freqStr); err == nil {
								gpu.ClockMemory = freq
							}
						}
						break
					}
				}
			}
		}

		gpus = append(gpus, gpu)
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("no AMD GPUs found")
	}

	return gpus, nil
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Find hwmon path for AMD GPU
func findAMDHwmonPath(cardPath string) string {
	hwmonBasePath := filepath.Join(cardPath, "hwmon")
	entries, err := os.ReadDir(hwmonBasePath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "hwmon") {
			return filepath.Join(hwmonBasePath, entry.Name())
		}
	}
	return ""
}

// GetOverclockSettings returns current overclock settings
func (r *LinuxReader) GetOverclockSettings(ctx context.Context, deviceID int) (*OverclockSettings, error) {
	// First check what type of GPU this is
	gpus, err := r.GetInfo(ctx)
	if err != nil || deviceID >= len(gpus) {
		return nil, fmt.Errorf("GPU device %d not found", deviceID)
	}

	gpu := gpus[deviceID]

	switch gpu.Vendor {
	case NVIDIA:
		return r.getNvidiaOverclockSettings(ctx, deviceID)
	case AMD:
		return r.getAMDOverclockSettings(ctx, deviceID)
	default:
		return nil, fmt.Errorf("overclocking not supported for %s GPUs", gpu.Vendor)
	}
}

// getNvidiaOverclockSettings gets NVIDIA GPU overclock settings
func (r *LinuxReader) getNvidiaOverclockSettings(ctx context.Context, deviceID int) (*OverclockSettings, error) {
	// Check if nvidia-settings is available
	cmd := exec.CommandContext(ctx, "nvidia-settings", "-q", fmt.Sprintf("[gpu:%d]/GPUGraphicsClockOffset[3]", deviceID))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-settings not available or GPU not found: %w", err)
	}

	settings := &OverclockSettings{}

	// Parse graphics clock offset
	if strings.Contains(string(output), "GPUGraphicsClockOffset") {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "GPUGraphicsClockOffset") && strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					offsetStr := strings.TrimSpace(parts[len(parts)-1])
					if offset, err := strconv.Atoi(offsetStr); err == nil {
						settings.CoreClockOffset = offset
					}
				}
			}
		}
	}

	// Get memory clock offset
	cmd = exec.CommandContext(ctx, "nvidia-settings", "-q", fmt.Sprintf("[gpu:%d]/GPUMemoryTransferRateOffset[3]", deviceID))
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "GPUMemoryTransferRateOffset") && strings.Contains(line, ":") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					offsetStr := strings.TrimSpace(parts[len(parts)-1])
					if offset, err := strconv.Atoi(offsetStr); err == nil {
						settings.MemoryClockOffset = offset
					}
				}
			}
		}
	}

	// Get power limit (from nvidia-smi)
	cmd = exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=power.limit", "--format=csv,noheader,nounits", "-i", fmt.Sprintf("%d", deviceID))
	if output, err := cmd.Output(); err == nil {
		powerStr := strings.TrimSpace(string(output))
		if power, err := strconv.ParseFloat(powerStr, 64); err == nil {
			settings.PowerLimit = int(power)
		}
	}

	return settings, nil
}

// getAMDOverclockSettings gets AMD GPU overclock settings
func (r *LinuxReader) getAMDOverclockSettings(ctx context.Context, deviceID int) (*OverclockSettings, error) {
	// Find the card path for this device
	cardPath := r.findAMDCardPath(deviceID)
	if cardPath == "" {
		return nil, fmt.Errorf("AMD GPU card%d not found", deviceID)
	}

	settings := &OverclockSettings{}

	// Get current GPU clock
	if sclkPath := filepath.Join(cardPath, "pp_dpm_sclk"); fileExists(sclkPath) {
		if sclkData, err := os.ReadFile(sclkPath); err == nil {
			lines := strings.Split(string(sclkData), "\n")
			for _, line := range lines {
				if strings.Contains(line, "*") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						freqStr := strings.TrimSuffix(parts[1], "Mhz")
						if _, err := strconv.Atoi(freqStr); err == nil {
							// AMD doesn't use offsets like NVIDIA, so we report current clock
							settings.CoreClockOffset = 0 // Will be base clock
						}
					}
					break
				}
			}
		}
	}

	// Get current memory clock
	if mclkPath := filepath.Join(cardPath, "pp_dpm_mclk"); fileExists(mclkPath) {
		if mclkData, err := os.ReadFile(mclkPath); err == nil {
			lines := strings.Split(string(mclkData), "\n")
			for _, line := range lines {
				if strings.Contains(line, "*") {
					settings.MemoryClockOffset = 0 // Will be base clock
					break
				}
			}
		}
	}

	// Get power limit (percentage)
	if powerCapPath := filepath.Join(cardPath, "power_dpm_force_performance_level"); fileExists(powerCapPath) {
		settings.PowerLimit = 100 // Default to 100% if we can't read specific value
	}

	return settings, nil
}

// findAMDCardPath finds the DRM card path for the given device ID
func (r *LinuxReader) findAMDCardPath(deviceID int) string {
	drmPath := "/sys/class/drm"
	entries, err := os.ReadDir(drmPath)
	if err != nil {
		return ""
	}

	currentID := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "card") || strings.Contains(entry.Name(), "-") {
			continue
		}

		cardPath := filepath.Join(drmPath, entry.Name(), "device")
		vendorPath := filepath.Join(cardPath, "vendor")

		if vendorData, err := os.ReadFile(vendorPath); err == nil {
			vendor := strings.TrimSpace(string(vendorData))
			if vendor == "0x1002" { // AMD vendor ID
				if currentID == deviceID {
					return cardPath
				}
				currentID++
			}
		}
	}
	return ""
}

// SetOverclockSettings applies overclock settings with detailed results
func (r *LinuxReader) SetOverclockSettings(ctx context.Context, deviceID int, settings *OverclockSettings) (*OverclockResult, error) {
	// First check what type of GPU this is
	gpus, err := r.GetInfo(ctx)
	if err != nil || deviceID >= len(gpus) {
		return nil, fmt.Errorf("GPU device %d not found", deviceID)
	}

	gpu := gpus[deviceID]

	switch gpu.Vendor {
	case NVIDIA:
		return r.setNvidiaOverclockSettings(ctx, deviceID, settings)
	case AMD:
		return r.setAMDOverclockSettings(ctx, deviceID, settings)
	default:
		return nil, fmt.Errorf("overclocking not supported for %s GPUs", gpu.Vendor)
	}
}

// setNvidiaOverclockSettings applies NVIDIA GPU overclock settings
func (r *LinuxReader) setNvidiaOverclockSettings(ctx context.Context, deviceID int, settings *OverclockSettings) (*OverclockResult, error) {
	result := &OverclockResult{
		Success:  true,
		Applied:  []string{},
		Warnings: []string{},
		Errors:   []string{},
	}

	// Validate device exists
	cmd := exec.CommandContext(ctx, "nvidia-smi", "-i", fmt.Sprintf("%d", deviceID))
	if _, err := cmd.Output(); err != nil {
		return nil, fmt.Errorf("GPU device %d not found", deviceID)
	}

	// Apply graphics clock offset
	if settings.CoreClockOffset != 0 {
		cmd := exec.CommandContext(ctx, "nvidia-settings",
			"-a", fmt.Sprintf("[gpu:%d]/GPUGraphicsClockOffset[3]=%d", deviceID, settings.CoreClockOffset))
		if err := cmd.Run(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to set core clock offset: %v", err))
			result.Success = false
		} else {
			result.Applied = append(result.Applied, fmt.Sprintf("core clock offset: %+d MHz", settings.CoreClockOffset))
		}
	}

	// Apply memory clock offset
	if settings.MemoryClockOffset != 0 {
		cmd := exec.CommandContext(ctx, "nvidia-settings",
			"-a", fmt.Sprintf("[gpu:%d]/GPUMemoryTransferRateOffset[3]=%d", deviceID, settings.MemoryClockOffset))
		if err := cmd.Run(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to set memory clock offset: %v", err))
			result.Success = false
		} else {
			result.Applied = append(result.Applied, fmt.Sprintf("memory clock offset: %+d MHz", settings.MemoryClockOffset))
		}
	}

	// Apply power limit (if supported)
	if settings.PowerLimit > 0 {
		cmd := exec.CommandContext(ctx, "nvidia-smi", "-i", fmt.Sprintf("%d", deviceID),
			"-pl", fmt.Sprintf("%d", settings.PowerLimit))
		if err := cmd.Run(); err != nil {
			// Check if it's a permission error
			if strings.Contains(err.Error(), "Insufficient permissions") ||
				strings.Contains(err.Error(), "permission denied") ||
				strings.Contains(err.Error(), "Operation not permitted") {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("power limit setting requires root privileges (requested: %d%%)", settings.PowerLimit))
			} else {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to set power limit: %v", err))
			}
		} else {
			result.Applied = append(result.Applied, fmt.Sprintf("power limit: %d%%", settings.PowerLimit))
		}
	}

	// Apply fan speed if specified
	if settings.FanSpeed > 0 {
		cmd := exec.CommandContext(ctx, "nvidia-settings",
			"-a", fmt.Sprintf("[gpu:%d]/GPUFanControlState=1", deviceID),
			"-a", fmt.Sprintf("[fan:%d]/GPUTargetFanSpeed=%d", deviceID, settings.FanSpeed))
		if err := cmd.Run(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to set fan speed: %v", err))
			result.Success = false
		} else {
			result.Applied = append(result.Applied, fmt.Sprintf("fan speed: %d%%", settings.FanSpeed))
		}
	}

	// If we have warnings but no errors, consider it a partial success
	if len(result.Errors) == 0 && len(result.Warnings) > 0 {
		result.Success = true
	}

	return result, nil
}

// setAMDOverclockSettings applies AMD GPU overclock settings
func (r *LinuxReader) setAMDOverclockSettings(ctx context.Context, deviceID int, settings *OverclockSettings) (*OverclockResult, error) {
	result := &OverclockResult{
		Success:  true,
		Applied:  []string{},
		Warnings: []string{},
		Errors:   []string{},
	}

	// Find the card path for this device
	cardPath := r.findAMDCardPath(deviceID)
	if cardPath == "" {
		return nil, fmt.Errorf("AMD GPU card%d not found", deviceID)
	}

	// AMD overclocking requires manual performance level
	perfLevelPath := filepath.Join(cardPath, "power_dpm_force_performance_level")
	if fileExists(perfLevelPath) {
		// Set to manual mode for overclocking
		if err := os.WriteFile(perfLevelPath, []byte("manual"), 0644); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to set manual performance level (may need root): %v", err))
		} else {
			result.Applied = append(result.Applied, "enabled manual performance mode")
		}
	}

	// Apply GPU clock settings
	if settings.CoreClockOffset != 0 {
		sclkPath := filepath.Join(cardPath, "pp_dpm_sclk")
		if fileExists(sclkPath) {
			// For AMD, we'll try to set the highest performance state
			// This is a simplified approach - real overclocking would need more complex logic
			if sclkData, err := os.ReadFile(sclkPath); err == nil {
				lines := strings.Split(string(sclkData), "\n")
				var maxState string
				for _, line := range lines {
					if strings.TrimSpace(line) != "" && !strings.Contains(line, "*") {
						parts := strings.Fields(line)
						if len(parts) >= 1 {
							maxState = parts[0]
						}
					}
				}
				if maxState != "" {
					sclkMaskPath := filepath.Join(cardPath, "pp_dpm_sclk")
					if err := os.WriteFile(sclkMaskPath, []byte(maxState), 0644); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("failed to set GPU clock state (may need root): %v", err))
					} else {
						result.Applied = append(result.Applied, fmt.Sprintf("set GPU to performance state %s", maxState))
					}
				}
			}
		}
	}

	// Apply memory clock settings
	if settings.MemoryClockOffset != 0 {
		mclkPath := filepath.Join(cardPath, "pp_dpm_mclk")
		if fileExists(mclkPath) {
			// Similar approach for memory clock
			if mclkData, err := os.ReadFile(mclkPath); err == nil {
				lines := strings.Split(string(mclkData), "\n")
				var maxState string
				for _, line := range lines {
					if strings.TrimSpace(line) != "" && !strings.Contains(line, "*") {
						parts := strings.Fields(line)
						if len(parts) >= 1 {
							maxState = parts[0]
						}
					}
				}
				if maxState != "" {
					mclkMaskPath := filepath.Join(cardPath, "pp_dpm_mclk")
					if err := os.WriteFile(mclkMaskPath, []byte(maxState), 0644); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("failed to set memory clock state (may need root): %v", err))
					} else {
						result.Applied = append(result.Applied, fmt.Sprintf("set memory to performance state %s", maxState))
					}
				}
			}
		}
	}

	// Power limit adjustment
	if settings.PowerLimit > 0 && settings.PowerLimit != 100 {
		result.Warnings = append(result.Warnings, "AMD power limit adjustment requires additional tools like 'amdgpu-pro' drivers")
	}

	// Fan speed control for AMD
	if settings.FanSpeed > 0 {
		// Look for hwmon fan controls
		hwmonPath := findAMDHwmonPath(cardPath)
		if hwmonPath != "" {
			pwm1Path := filepath.Join(hwmonPath, "pwm1")
			pwm1EnablePath := filepath.Join(hwmonPath, "pwm1_enable")

			if fileExists(pwm1Path) && fileExists(pwm1EnablePath) {
				// Set manual fan control
				if err := os.WriteFile(pwm1EnablePath, []byte("1"), 0644); err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("failed to enable manual fan control: %v", err))
				} else {
					// Convert percentage to PWM value (0-255)
					pwmVal := (settings.FanSpeed * 255) / 100
					if err := os.WriteFile(pwm1Path, []byte(fmt.Sprintf("%d", pwmVal)), 0644); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("failed to set fan speed: %v", err))
					} else {
						result.Applied = append(result.Applied, fmt.Sprintf("fan speed: %d%%", settings.FanSpeed))
					}
				}
			}
		}
	}

	// Check if any operations succeeded
	if len(result.Applied) == 0 && len(result.Warnings) > 0 {
		result.Warnings = append(result.Warnings, "AMD GPU overclocking may require root privileges or additional driver setup")
	}

	return result, nil
}
