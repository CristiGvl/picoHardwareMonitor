package platform

import (
	"fmt"
	"runtime"
)

// SupportedOS represents supported operating systems
type SupportedOS string

const (
	Linux   SupportedOS = "linux"
	Windows SupportedOS = "windows"
)

// GetOS returns the current operating system
func GetOS() SupportedOS {
	return SupportedOS(runtime.GOOS)
}

// IsSupported returns true if the current OS is supported
func IsSupported() bool {
	os := GetOS()
	return os == Linux || os == Windows
}

// ValidateSupport returns an error if the current OS is not supported
func ValidateSupport() error {
	if !IsSupported() {
		return fmt.Errorf("unsupported operating system: %s. Supported: linux, windows", runtime.GOOS)
	}
	return nil
}
