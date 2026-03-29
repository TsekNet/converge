//go:build windows

// Package winreg provides shared Windows registry helpers used by both
// the registry extension and registry conditions.
package winreg

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ParseKeyPath splits "HKLM\Software\..." into a root handle and subkey path.
// Supported root abbreviations: HKLM, HKCU, HKCR, HKU, HKCC and their long forms.
func ParseKeyPath(full string) (registry.Key, string, error) {
	idx := strings.IndexByte(full, '\\')
	if idx < 0 {
		return 0, "", fmt.Errorf("invalid registry path %q: missing root hive", full)
	}
	rootStr, path := full[:idx], full[idx+1:]
	switch strings.ToUpper(rootStr) {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		return registry.LOCAL_MACHINE, path, nil
	case "HKCU", "HKEY_CURRENT_USER":
		return registry.CURRENT_USER, path, nil
	case "HKCR", "HKEY_CLASSES_ROOT":
		return registry.CLASSES_ROOT, path, nil
	case "HKU", "HKEY_USERS":
		return registry.USERS, path, nil
	case "HKCC", "HKEY_CURRENT_CONFIG":
		return registry.CURRENT_CONFIG, path, nil
	default:
		return 0, "", fmt.Errorf("unknown registry root %q", rootStr)
	}
}
