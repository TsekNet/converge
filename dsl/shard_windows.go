//go:build windows

package dsl

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

// detectSerial reads the BIOS serial number from the Windows registry.
// Windows populates HKLM\HARDWARE\DESCRIPTION\System\BIOS from SMBIOS
// data at boot. This avoids GetSystemFirmwareTable (requires unsafe) and
// PowerShell (exec). Called once per process, cached via sync.Once.
func detectSerial() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\BIOS`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()

	val, _, err := k.GetStringValue("SerialNumber")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}
