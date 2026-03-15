//go:build windows

package dsl

import (
	"strings"

	"golang.org/x/sys/windows"
)

// detectSerial reads the BIOS serial number via the Win32
// GetSystemFirmwareTable API (SMBIOS type 1, System Information).
func detectSerial() string {
	// RSMB = Raw SMBIOS data provider.
	const rsmb = 0x52534D42

	// First call: get required buffer size.
	size, _ := windows.GetSystemFirmwareTable(rsmb, 0, nil, 0)
	if size == 0 {
		return ""
	}

	buf := make([]byte, size)
	n, _ := windows.GetSystemFirmwareTable(rsmb, 0, buf, size)
	if n == 0 {
		return ""
	}

	return parseSMBIOSSerial(buf[:n])
}

// parseSMBIOSSerial walks the raw SMBIOS table looking for Type 1
// (System Information) and extracts the Serial Number string.
func parseSMBIOSSerial(data []byte) string {
	if len(data) < 8 {
		return ""
	}
	// Skip the 8-byte RawSMBIOSData header (used by Windows).
	tableData := data[8:]
	return walkSMBIOS(tableData)
}

func walkSMBIOS(data []byte) string {
	offset := 0
	for offset < len(data)-4 {
		sType := data[offset]
		sLen := int(data[offset+1])
		if sLen < 4 || offset+sLen > len(data) {
			break
		}

		if sType == 1 && sLen >= 0x19 {
			// Type 1: System Information. Serial Number is string index at offset 7.
			serialIdx := int(data[offset+7])
			return smbiosString(data[offset+sLen:], serialIdx)
		}

		// Skip formatted area.
		strStart := offset + sLen
		// Walk past unformatted (string) area: terminated by double NUL.
		i := strStart
		for i < len(data)-1 {
			if data[i] == 0 && data[i+1] == 0 {
				offset = i + 2
				break
			}
			i++
		}
		if i >= len(data)-1 {
			break
		}
	}
	return ""
}

// smbiosString extracts the Nth (1-based) NUL-terminated string from
// the unformatted area of an SMBIOS structure.
func smbiosString(data []byte, index int) string {
	if index <= 0 {
		return ""
	}
	current := 1
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			if current == index {
				s := string(data[start:i])
				return strings.TrimSpace(s)
			}
			current++
			start = i + 1
			// Double NUL = end of string area.
			if i+1 < len(data) && data[i+1] == 0 {
				break
			}
		}
	}
	return ""
}
