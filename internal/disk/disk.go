package disk

import "context"

// Info represents disk information
type Info struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Filesystem string  `json:"filesystem"`
	Total      uint64  `json:"total_mb"`
	Used       uint64  `json:"used_mb"`
	Available  uint64  `json:"available_mb"`
	Usage      float64 `json:"usage_percent"`
}

// Reader interface for disk monitoring
type Reader interface {
	GetInfo(ctx context.Context) ([]*Info, error)
}

// NewReader creates a new disk reader for the current platform
func NewReader() Reader {
	return newPlatformReader()
}
