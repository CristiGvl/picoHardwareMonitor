package fan

import "context"

// FanMode represents different fan control modes
type FanMode string

const (
	ModeAuto  FanMode = "auto"
	ModeFixed FanMode = "fixed"
	ModeCurve FanMode = "curve"
)

// CurvePoint represents a point in a fan curve
type CurvePoint struct {
	Temperature int `json:"temperature_celsius"`
	FanSpeed    int `json:"fan_speed_percent"`
}

// Settings represents fan control settings
type Settings struct {
	Mode       FanMode      `json:"mode"`
	FixedSpeed int          `json:"fixed_speed_percent,omitempty"`
	Curve      []CurvePoint `json:"curve,omitempty"`
}

// Info represents fan information
type Info struct {
	Name   string `json:"name"`
	RPM    int    `json:"rpm"`
	Speed  int    `json:"speed_percent"`
	MaxRPM int    `json:"max_rpm"`
}

// Controller interface for fan control
type Controller interface {
	GetFans(ctx context.Context) ([]*Info, error)
	GetSettings(ctx context.Context, fanID int) (*Settings, error)
	SetSettings(ctx context.Context, fanID int, settings *Settings) error
}

// NewController creates a new fan controller for the current platform
func NewController() Controller {
	return newPlatformController()
}
