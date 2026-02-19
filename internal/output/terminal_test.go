package output

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0ms"},
		{"milliseconds small", 12 * time.Millisecond, "12ms"},
		{"milliseconds large", 500 * time.Millisecond, "500ms"},
		{"seconds with decimal", 1200 * time.Millisecond, "1.2s"},
		{"seconds rounded", 3456 * time.Millisecond, "3.5s"},
		{"exactly one second", 1000 * time.Millisecond, "1.0s"},
		{"sub-millisecond", 500 * time.Microsecond, "0ms"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestTerminalPrinter_Dots(t *testing.T) {
	tests := []struct {
		name       string
		maxNameLen int
		resource   string
		wantMinLen int
	}{
		{"normal padding", 20, "File /etc/motd", 5},
		{"long name exceeds max", 5, "very long resource name that exceeds max", 4},
		{"exact max length", 14, "File /etc/motd", 4},
		{"zero max", 0, "File /etc/motd", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TerminalPrinter{maxNameLen: tt.maxNameLen}
			dots := p.dots(tt.resource)
			if len(dots) < tt.wantMinLen {
				t.Errorf("dots(%q) len = %d, want >= %d", tt.resource, len(dots), tt.wantMinLen)
			}
		})
	}
}

func TestSerialPrinter_Dots(t *testing.T) {
	tests := []struct {
		name       string
		maxNameLen int
		resource   string
		wantMinLen int
	}{
		{"normal padding", 20, "File /etc/motd", 5},
		{"long name", 5, "very long name", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &SerialPrinter{maxNameLen: tt.maxNameLen}
			dots := p.dots(tt.resource)
			if len(dots) < tt.wantMinLen {
				t.Errorf("dots(%q) len = %d, want >= %d", tt.resource, len(dots), tt.wantMinLen)
			}
		})
	}
}

func TestNewPrinters(t *testing.T) {
	tests := []struct {
		name    string
		printer Printer
	}{
		{"terminal", NewTerminalPrinter()},
		{"serial", NewSerialPrinter()},
		{"json", NewJSONPrinter()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.printer == nil {
				t.Fatalf("New%sPrinter() returned nil", tt.name)
			}
		})
	}
}
