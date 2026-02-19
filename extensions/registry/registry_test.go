package registry

import (
	"context"
	"runtime"
	"testing"
)

func TestRegistry_IDAndString(t *testing.T) {
	tests := []struct {
		key     string
		wantID  string
		wantStr string
	}{
		{`HKLM\SOFTWARE\Test`, `registry:HKLM\SOFTWARE\Test`, `Registry HKLM\SOFTWARE\Test`},
		{`HKCU\Control Panel`, `registry:HKCU\Control Panel`, `Registry HKCU\Control Panel`},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			r := New(tt.key)
			if r.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", r.ID(), tt.wantID)
			}
			if r.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", r.String(), tt.wantStr)
			}
		})
	}
}

func TestRegistry_IsCritical(t *testing.T) {
	r := New(`HKLM\SOFTWARE\Test`)
	if r.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	r.Critical = true
	if !r.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}

func TestRegistry_StubBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub tests are for non-Windows")
	}

	ctx := context.Background()
	r := New(`HKLM\SOFTWARE\Test`)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"check returns in sync", func() error {
			state, err := r.Check(ctx)
			if err != nil {
				return err
			}
			if !state.InSync {
				t.Error("stub Check should return InSync=true")
			}
			return nil
		}},
		{"apply returns not changed", func() error {
			result, err := r.Apply(ctx)
			if err != nil {
				return err
			}
			if result.Changed {
				t.Error("stub Apply should return Changed=false")
			}
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
