package gpu

import "context"

// Vendor represents GPU vendor
type Vendor string

const (
	NVIDIA  Vendor = "nvidia"
	AMD     Vendor = "amd"
	Intel   Vendor = "intel"
	Unknown Vendor = "unknown"
)

// Info represents GPU information
type Info struct {
	Vendor      Vendor  `json:"vendor"`
	Model       string  `json:"model"`
	VRAM        uint64  `json:"vram_mb"`
	Usage       float64 `json:"usage_percent"`
	MemoryUsage float64 `json:"memory_usage_percent"`
	Temperature float64 `json:"temperature_celsius"`
	PowerUsage  float64 `json:"power_usage_watts"`
	ClockCore   int     `json:"clock_core_mhz"`
	ClockMemory int     `json:"clock_memory_mhz"`
}

// OverclockSettings represents GPU overclocking settings
type OverclockSettings struct {
	CoreClockOffset   int `json:"core_clock_offset_mhz"`
	MemoryClockOffset int `json:"memory_clock_offset_mhz"`
	PowerLimit        int `json:"power_limit_percent"`
	FanSpeed          int `json:"fan_speed_percent"`
}

// OverclockResult represents the result of an overclocking operation
type OverclockResult struct {
	Success  bool     `json:"success"`
	Applied  []string `json:"applied"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
}

// Reader interface for GPU monitoring
type Reader interface {
	GetInfo(ctx context.Context) ([]*Info, error)
	GetOverclockSettings(ctx context.Context, deviceID int) (*OverclockSettings, error)
	SetOverclockSettings(ctx context.Context, deviceID int, settings *OverclockSettings) (*OverclockResult, error)
}

// NewReader creates a new GPU reader for the current platform
func NewReader() Reader {
	return newPlatformReader()
}
