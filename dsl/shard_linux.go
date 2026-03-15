//go:build linux

package dsl

import (
	"os"
	"strings"
)

// detectSerial reads the hardware serial from sysfs (DMI product serial).
// Requires root or relaxed sysfs permissions.
func detectSerial() string {
	data, err := os.ReadFile("/sys/class/dmi/id/product_serial")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
