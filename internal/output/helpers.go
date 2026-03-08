package output

import (
	"os"
	"strings"
)

// SupportsColor returns true if stdout is a TTY with ANSI support.
// On Windows, it attempts to enable virtual terminal processing first.
// Returns false when NO_COLOR is set, stdout is not a TTY, or the
// console doesn't support escape sequences (legacy PowerShell/cmd).
func SupportsColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if !isTTY() {
		return false
	}
	return enableVT()
}

// splitResource extracts the type and short name from an Extension's String().
// "File /etc/motd" -> ("File", "/etc/motd")
// "Package git"    -> ("Package", "git")
func splitResource(s string) (resType, resName string) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return s, ""
}

// capitalizeStatus uppercases the first letter of a status message for display.
func capitalizeStatus(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
