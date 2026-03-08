package secpol

import (
	"context"
	"runtime"
	"testing"
)

func TestSecurityPolicy_IDAndString(t *testing.T) {
	tests := []struct {
		category, key string
		wantID        string
		wantStr       string
	}{
		{"password", "MinimumPasswordLength", "secpol:password:MinimumPasswordLength", "SecurityPolicy password/MinimumPasswordLength"},
		{"lockout", "LockoutThreshold", "secpol:lockout:LockoutThreshold", "SecurityPolicy lockout/LockoutThreshold"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			s := New(tt.category, tt.key, "14")
			if s.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", s.ID(), tt.wantID)
			}
			if s.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", s.String(), tt.wantStr)
			}
		})
	}
}

func TestSecurityPolicy_IsCritical(t *testing.T) {
	s := New("password", "MinimumPasswordLength", "14")
	if s.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	s.Critical = true
	if !s.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}

func TestSecurityPolicy_New(t *testing.T) {
	s := New("lockout", "LockoutThreshold", "5")
	if s.Category != "lockout" {
		t.Errorf("Category = %q, want %q", s.Category, "lockout")
	}
	if s.Key != "LockoutThreshold" {
		t.Errorf("Key = %q, want %q", s.Key, "LockoutThreshold")
	}
	if s.Value != "5" {
		t.Errorf("Value = %q, want %q", s.Value, "5")
	}
}

func TestSecurityPolicy_StubBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub tests are for non-Windows")
	}

	ctx := context.Background()
	s := New("password", "MinimumPasswordLength", "14")

	t.Run("check returns in sync", func(t *testing.T) {
		state, err := s.Check(ctx)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if !state.InSync {
			t.Error("stub Check should return InSync=true")
		}
	})

	t.Run("apply returns not changed", func(t *testing.T) {
		result, err := s.Apply(ctx)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if result.Changed {
			t.Error("stub Apply should return Changed=false")
		}
	})
}
