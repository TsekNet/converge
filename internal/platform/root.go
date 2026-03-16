package platform

import (
	"os"
	"runtime"
)

// IsRoot returns true if the process has root (Unix) or administrator (Windows) privileges.
func IsRoot() bool {
	if runtime.GOOS == "windows" {
		f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		if err != nil {
			return false
		}
		f.Close()
		return true
	}
	return os.Geteuid() == 0
}
