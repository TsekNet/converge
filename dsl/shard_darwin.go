//go:build darwin

package dsl

import "golang.org/x/sys/unix"

// detectSerial returns the machine's hardware UUID via sysctl.
// macOS does not expose the platform serial number via sysctl, and reading
// it via IOKit requires CGo + unsafe. The hardware UUID (kern.uuid) is
// unique per machine, always present, and serves the same sharding purpose.
func detectSerial() string {
	uuid, err := unix.SysctlbyName("kern.uuid")
	if err != nil {
		return ""
	}
	return uuid
}
