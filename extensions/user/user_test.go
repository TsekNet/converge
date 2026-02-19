package user

import (
	"context"
	"os/user"
	"testing"
)

func TestUser_IDAndString(t *testing.T) {
	tests := []struct {
		name    string
		wantID  string
		wantStr string
	}{
		{"devuser", "user:devuser", "User devuser"},
		{"admin", "user:admin", "User admin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := New(tt.name, nil, "")
			if u.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", u.ID(), tt.wantID)
			}
			if u.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", u.String(), tt.wantStr)
			}
		})
	}
}

func TestUser_Check_ExistingUser(t *testing.T) {
	ctx := context.Background()

	current, err := user.Current()
	if err != nil {
		t.Skip("cannot determine current user")
	}

	u := New(current.Username, nil, "")
	state, err := u.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !state.InSync {
		t.Logf("current user not fully in sync: %+v (expected for no shell specified)", state.Changes)
	}
}

func TestUser_Check_NonexistentUser(t *testing.T) {
	ctx := context.Background()

	u := New("converge-test-user-does-not-exist-xyz", []string{"sudo"}, "/bin/bash")
	state, err := u.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if state.InSync {
		t.Error("nonexistent user should not be in sync")
	}
	if len(state.Changes) < 1 {
		t.Error("should have at least one change (create user)")
	}
}

func TestUser_Check_WithShell(t *testing.T) {
	ctx := context.Background()

	current, err := user.Current()
	if err != nil {
		t.Skip("cannot determine current user")
	}

	tests := []struct {
		name  string
		shell string
	}{
		{"matching shell", "/bin/bash"},
		{"different shell", "/usr/bin/zsh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := New(current.Username, nil, tt.shell)
			state, err := u.Check(ctx)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			t.Logf("InSync=%v Changes=%+v", state.InSync, state.Changes)
		})
	}
}

func TestUser_IsCritical(t *testing.T) {
	u := New("devuser", nil, "")
	if u.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	u.Critical = true
	if !u.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}

func TestUser_New(t *testing.T) {
	u := New("admin", []string{"sudo", "docker"}, "/bin/zsh")
	if u.Name != "admin" {
		t.Errorf("Name = %q, want %q", u.Name, "admin")
	}
	if len(u.Groups) != 2 {
		t.Errorf("Groups len = %d, want 2", len(u.Groups))
	}
	if u.Shell != "/bin/zsh" {
		t.Errorf("Shell = %q, want %q", u.Shell, "/bin/zsh")
	}
}

func TestUser_Apply_NonexistentUser(t *testing.T) {
	ctx := context.Background()
	u := New("converge-test-user-does-not-exist-xyz", []string{"sudo"}, "/bin/bash")
	_, err := u.Apply(ctx)
	if err == nil {
		t.Log("Apply may succeed if running as root, or fail otherwise")
	}
}
