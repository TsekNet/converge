package auditpol

import (
	"context"
	"runtime"
	"testing"
)

func TestAuditPolicy_IDAndString(t *testing.T) {
	tests := []struct {
		subcategory string
		wantID      string
		wantStr     string
	}{
		{"Credential Validation", "auditpol:Credential Validation", "AuditPolicy Credential Validation"},
		{"Logon", "auditpol:Logon", "AuditPolicy Logon"},
		{"Process Creation", "auditpol:Process Creation", "AuditPolicy Process Creation"},
	}
	for _, tt := range tests {
		t.Run(tt.subcategory, func(t *testing.T) {
			a := New(tt.subcategory, true, true)
			if a.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", a.ID(), tt.wantID)
			}
			if a.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", a.String(), tt.wantStr)
			}
		})
	}
}

func TestAuditPolicy_IsCritical(t *testing.T) {
	a := New("Logon", true, false)
	if a.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	a.Critical = true
	if !a.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}

func TestAuditPolicy_New(t *testing.T) {
	a := New("Credential Validation", true, false)
	if a.Subcategory != "Credential Validation" {
		t.Errorf("Subcategory = %q, want %q", a.Subcategory, "Credential Validation")
	}
	if !a.Success {
		t.Error("Success should be true")
	}
	if a.Failure {
		t.Error("Failure should be false")
	}
}

func TestAuditPolicy_StubBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub tests are for non-Windows")
	}

	ctx := context.Background()
	a := New("Logon", true, true)

	t.Run("check returns in sync", func(t *testing.T) {
		state, err := a.Check(ctx)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !state.InSync {
			t.Error("stub Check should return InSync=true")
		}
	})

	t.Run("apply returns not changed", func(t *testing.T) {
		result, err := a.Apply(ctx)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if result.Changed {
			t.Error("stub Apply should return Changed=false")
		}
	})
}
