//go:build darwin

package dsl

import (
	"os/exec"
	"strings"
)

// detectSerial returns the machine's hardware UUID via sysctl.
// macOS does not expose the platform serial number via sysctl, and reading
// it via IOKit requires CGo + unsafe. The hardware UUID (kern.uuid) is
// unique per machine, always present, and serves the same sharding purpose.
//
// Uses /usr/sbin/sysctl instead of golang.org/x/sys/unix.SysctlbyName
// because the latter is not available during CGO_ENABLED=0 cross-compilation.
func detectSerial() string {
	out, err := exec.Command("/usr/sbin/sysctl", "-n", "kern.uuid").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
