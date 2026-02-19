package output

import "strings"

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
