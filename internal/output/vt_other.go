//go:build !windows

package output

// enableVT is a no-op on non-Windows platforms -- terminals natively support ANSI.
func enableVT() bool { return true }
