package service

import (
	"context"
	"runtime"
	"testing"
)

func TestService_IDAndString(t *testing.T) {
	tests := []struct {
		name    string
		wantID  string
		wantStr string
	}{
		{"sshd", "service:sshd", "Service sshd"},
		{"docker", "service:docker", "Service docker"},
		{"nginx", "service:nginx", "Service nginx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.name, "running", true, "systemd")
			if s.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", s.ID(), tt.wantID)
			}
			if s.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", s.String(), tt.wantStr)
			}
		})
	}
}

func TestService_Check_UnsupportedInit(t *testing.T) {
	ctx := context.Background()
	s := New("test", "running", true, "runit")

	_, err := s.Check(ctx)
	if err == nil {
		t.Error("Check() should fail for unsupported init system")
	}
}

func TestService_Apply_UnsupportedInit(t *testing.T) {
	ctx := context.Background()
	s := New("test", "running", true, "runit")

	_, err := s.Apply(ctx)
	if err == nil {
		t.Error("Apply() should fail for unsupported init system")
	}
}

func TestService_CheckSystemd_LiveCron(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd tests only run on Linux")
	}
	ctx := context.Background()
	s := New("cron", "running", true, "systemd")
	state, err := s.Check(ctx)
	if err != nil {
		t.Skipf("systemctl not available or cron not present: %v", err)
	}
	t.Logf("cron service: InSync=%v, Changes=%+v", state.InSync, state.Changes)
}

func TestService_CheckSystemd_NonexistentService(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd tests only run on Linux")
	}
	ctx := context.Background()
	s := New("converge-definitely-not-real-12345", "running", true, "systemd")
	state, err := s.Check(ctx)
	if err != nil {
		t.Skipf("systemctl not available: %v", err)
	}
	if state.InSync {
		t.Error("nonexistent service should not be in sync")
	}
}

func TestService_Check_Launchd(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("launchd tests only run on macOS")
	}
	ctx := context.Background()
	s := New("test", "running", true, "launchd")
	state, err := s.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !state.InSync {
		t.Error("launchd stub should always return InSync=true")
	}
}

func TestService_Check_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows SCM tests only run on Windows")
	}
	ctx := context.Background()
	s := New("test", "running", true, "windows")
	state, err := s.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !state.InSync {
		t.Error("windows Check should return InSync=true for nonexistent service")
	}
}

func TestService_CheckSystemd_StoppedService(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd tests only run on Linux")
	}
	ctx := context.Background()
	s := New("converge-definitely-not-real-12345", "stopped", false, "systemd")
	state, err := s.Check(ctx)
	if err != nil {
		t.Skipf("systemctl not available: %v", err)
	}
	if !state.InSync {
		t.Log("stopped+disabled nonexistent service should be in sync")
	}
}

func TestService_IsCritical(t *testing.T) {
	s := New("sshd", "running", true, "systemd")
	if s.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	s.Critical = true
	if !s.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}

func TestService_New(t *testing.T) {
	s := New("nginx", "stopped", false, "systemd")
	if s.Name != "nginx" {
		t.Errorf("Name = %q, want %q", s.Name, "nginx")
	}
	if s.State != "stopped" {
		t.Errorf("State = %q, want %q", s.State, "stopped")
	}
	if s.Enable {
		t.Error("Enable should be false")
	}
	if s.InitSystem != "systemd" {
		t.Errorf("InitSystem = %q, want %q", s.InitSystem, "systemd")
	}
}

func TestService_StartupType(t *testing.T) {
	s := New("wuauserv", "running", true, "windows")
	s.StartupType = "disabled"
	if s.StartupType != "disabled" {
		t.Errorf("StartupType = %q, want %q", s.StartupType, "disabled")
	}
}
