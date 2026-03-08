package sysctl

import (
	"runtime"
	"testing"
)

func TestSysctl_ID(t *testing.T) {
	s := New("net.ipv4.ip_forward", "0")
	if got := s.ID(); got != "sysctl:net.ipv4.ip_forward" {
		t.Errorf("ID() = %q, want %q", got, "sysctl:net.ipv4.ip_forward")
	}
}

func TestSysctl_String(t *testing.T) {
	s := New("net.ipv4.ip_forward", "0")
	if got := s.String(); got != "Sysctl net.ipv4.ip_forward = 0" {
		t.Errorf("String() = %q, want %q", got, "Sysctl net.ipv4.ip_forward = 0")
	}
}

func TestSysctl_IsCritical(t *testing.T) {
	s := New("net.ipv4.ip_forward", "0")
	if s.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	s.Critical = true
	if !s.IsCritical() {
		t.Error("IsCritical() should be true after setting")
	}
}

func TestSysctl_New_Defaults(t *testing.T) {
	s := New("kernel.randomize_va_space", "2")
	if s.Key != "kernel.randomize_va_space" {
		t.Errorf("Key = %q", s.Key)
	}
	if s.Value != "2" {
		t.Errorf("Value = %q", s.Value)
	}
	if !s.Persist {
		t.Error("Persist should default to true")
	}
}

func TestSysctl_Check_ReadOnly(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysctl Check requires /proc/sys (linux only)")
	}

	s := New("kernel.ostype", "Linux")
	state, err := s.Check(nil)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if !state.InSync {
		t.Error("kernel.ostype should be 'Linux' on a Linux system")
	}
}
