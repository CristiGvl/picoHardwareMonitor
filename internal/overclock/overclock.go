package overclock

import "context"

// Settings represents overclocking settings
type Settings struct {
	DeviceID          int     `json:"device_id"`
	CoreClockOffset   int     `json:"core_clock_offset_mhz"`
	MemoryClockOffset int     `json:"memory_clock_offset_mhz"`
	PowerLimit        int     `json:"power_limit_percent"`
	TempLimit         int     `json:"temp_limit_celsius"`
	FanSpeed          int     `json:"fan_speed_percent"`
	VoltageOffset     float64 `json:"voltage_offset_mv"`
}

// Profile represents an overclocking profile
type Profile struct {
	Name     string    `json:"name"`
	Settings *Settings `json:"settings"`
}

// Controller interface for overclocking control
type Controller interface {
	GetSettings(ctx context.Context, deviceID int) (*Settings, error)
	SetSettings(ctx context.Context, settings *Settings) error
	GetProfiles(ctx context.Context) ([]*Profile, error)
	SaveProfile(ctx context.Context, profile *Profile) error
	LoadProfile(ctx context.Context, profileName string) error
}

// NewController creates a new overclocking controller for the current platform
func NewController() Controller {
	return newPlatformController()
}
