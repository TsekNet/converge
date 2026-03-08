package registry

import (
	"context"
	"runtime"
	"testing"
)

func TestRegistry_IDAndString(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		wantID  string
		wantStr string
	}{
		{`HKLM\SOFTWARE\Test`, "Foo", `registry:HKLM\SOFTWARE\Test\Foo`, `Registry HKLM\SOFTWARE\Test\Foo`},
		{`HKCU\Control Panel`, "Bar", `registry:HKCU\Control Panel\Bar`, `Registry HKCU\Control Panel\Bar`},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			r := New(tt.key)
			r.Value = tt.value
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

func TestRegistry_DefaultState(t *testing.T) {
	r := New(`HKLM\SOFTWARE\Test`)
	if r.State != "present" {
		t.Errorf("default State = %q, want %q", r.State, "present")
	}
}

func TestRegistry_StubBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub tests are for non-Windows")
	}

	ctx := context.Background()
	r := New(`HKLM\SOFTWARE\Test`)
	r.Value = "TestVal"

	t.Run("check returns in sync", func(t *testing.T) {
		state, err := r.Check(ctx)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !state.InSync {
			t.Error("stub Check should return InSync=true")
		}
	})

	t.Run("apply returns not changed", func(t *testing.T) {
		result, err := r.Apply(ctx)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if result.Changed {
			t.Error("stub Apply should return Changed=false")
		}
	})
}
