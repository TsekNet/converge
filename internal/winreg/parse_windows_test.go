//go:build windows

package winreg

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestParseKeyPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantKey registry.Key
		wantSub string
		wantErr bool
	}{
		{"HKLM short", `HKLM\SOFTWARE\Test`, registry.LOCAL_MACHINE, `SOFTWARE\Test`, false},
		{"HKLM long", `HKEY_LOCAL_MACHINE\SOFTWARE\Test`, registry.LOCAL_MACHINE, `SOFTWARE\Test`, false},
		{"HKCU short", `HKCU\Software\Test`, registry.CURRENT_USER, `Software\Test`, false},
		{"HKCU long", `HKEY_CURRENT_USER\Software`, registry.CURRENT_USER, `Software`, false},
		{"HKCR short", `HKCR\.txt`, registry.CLASSES_ROOT, `.txt`, false},
		{"HKCR long", `HKEY_CLASSES_ROOT\.txt`, registry.CLASSES_ROOT, `.txt`, false},
		{"HKU short", `HKU\.DEFAULT`, registry.USERS, `.DEFAULT`, false},
		{"HKU long", `HKEY_USERS\.DEFAULT`, registry.USERS, `.DEFAULT`, false},
		{"HKCC short", `HKCC\System`, registry.CURRENT_CONFIG, `System`, false},
		{"HKCC long", `HKEY_CURRENT_CONFIG\System`, registry.CURRENT_CONFIG, `System`, false},
		{"case insensitive", `hklm\SOFTWARE`, registry.LOCAL_MACHINE, `SOFTWARE`, false},
		{"missing backslash", `HKLM`, 0, "", true},
		{"unknown root", `BOGUS\Path`, 0, "", true},
		{"empty string", ``, 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			key, sub, err := ParseKeyPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKeyPath(%q) error: %v", tt.input, err)
			}
			if key != tt.wantKey {
				t.Errorf("root key = %v, want %v", key, tt.wantKey)
			}
			if sub != tt.wantSub {
				t.Errorf("subkey = %q, want %q", sub, tt.wantSub)
			}
		})
	}
}
