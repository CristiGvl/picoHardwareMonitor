package temps

import "context"

// Sensor represents a temperature sensor
type Sensor struct {
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Temperature float64 `json:"temperature_celsius"`
	Critical    float64 `json:"critical_celsius"`
	Max         float64 `json:"max_celsius"`
}

// Info represents temperature information
type Info struct {
	CPU    []*Sensor `json:"cpu"`
	GPU    []*Sensor `json:"gpu"`
	System []*Sensor `json:"system"`
	Drives []*Sensor `json:"drives"`
}

// Reader interface for temperature monitoring
type Reader interface {
	GetInfo(ctx context.Context) (*Info, error)
}

// NewReader creates a new temperature reader for the current platform
func NewReader() Reader {
	return newPlatformReader()
}
