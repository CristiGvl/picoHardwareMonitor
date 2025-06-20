package cpu

import "context"

// Info represents CPU information
type Info struct {
	Model     string  `json:"model"`
	Cores     int     `json:"cores"`
	Threads   int     `json:"threads"`
	Usage     float64 `json:"usage_percent"`
	Frequency float64 `json:"frequency_mhz"`
}

// Reader interface for CPU monitoring
type Reader interface {
	GetInfo(ctx context.Context) (*Info, error)
	GetUsage(ctx context.Context) (float64, error)
}

// NewReader creates a new CPU reader for the current platform
func NewReader() Reader {
	return newPlatformReader()
}
