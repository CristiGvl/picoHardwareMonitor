package memory

import "context"

// Info represents memory information
type Info struct {
	Total     uint64  `json:"total_mb"`
	Used      uint64  `json:"used_mb"`
	Available uint64  `json:"available_mb"`
	Usage     float64 `json:"usage_percent"`
}

// Reader interface for memory monitoring
type Reader interface {
	GetInfo(ctx context.Context) (*Info, error)
}

// NewReader creates a new memory reader for the current platform
func NewReader() Reader {
	return newPlatformReader()
}
