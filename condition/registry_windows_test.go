//go:build windows

package condition

import (
	"testing"
)

func TestRegistryCondition_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cond *registryCondition
		want string
	}{
		{
			"key exists",
			RegistryKeyExists(`HKLM\SOFTWARE\Test`),
			`registry key exists HKLM\SOFTWARE\Test`,
		},
		{
			"value exists",
			RegistryValueExists(`HKLM\SOFTWARE\Test`, "Version"),
			`registry value exists HKLM\SOFTWARE\Test\Version`,
		},
		{
			"value equals string",
			RegistryValueEquals(`HKLM\SOFTWARE\Test`, "Mode", "active"),
			`registry value HKLM\SOFTWARE\Test\Mode = active`,
		},
		{
			"value equals int",
			RegistryValueEquals(`HKLM\SOFTWARE\Test`, "Count", 42),
			`registry value HKLM\SOFTWARE\Test\Count = 42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cond.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRegistryCondition_Met_keyAbsent(t *testing.T) {
	t.Parallel()

	// A non-existent key should return false, nil (not met, no error).
	c := RegistryKeyExists(`HKLM\SOFTWARE\ConvergeTestNonExistent_` + t.Name())
	met, err := c.Met(nil)
	if err != nil {
		t.Fatalf("Met() error: %v", err)
	}
	if met {
		t.Error("Met() = true for non-existent key, want false")
	}
}

func TestRegistryCondition_Met_invalidPath(t *testing.T) {
	t.Parallel()

	c := RegistryKeyExists("BOGUS_ROOT")
	_, err := c.Met(nil)
	if err == nil {
		t.Fatal("expected error for invalid registry path, got nil")
	}
}

func TestRegistryCondition_Met_knownKey(t *testing.T) {
	t.Parallel()

	// HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion exists on all Windows machines.
	c := RegistryKeyExists(`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion`)
	met, err := c.Met(nil)
	if err != nil {
		t.Fatalf("Met() error: %v", err)
	}
	if !met {
		t.Error("Met() = false for known existing key, want true")
	}
}
