package secpol

import "testing"

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
