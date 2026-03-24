//go:build linux

package dsl

import "testing"

func TestRun_Sysctl(t *testing.T) {
	t.Parallel()

	app := New()
	run := newRun(app)
	run.Sysctl("net.ipv4.ip_forward", SysctlOpts{Value: "1"})

	if run.Err() != nil {
		t.Fatalf("Sysctl() set error: %v", run.Err())
	}

	resources := run.Resources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if got := resources[0].ID(); got != "sysctl:net.ipv4.ip_forward" {
		t.Errorf("ID() = %q, want %q", got, "sysctl:net.ipv4.ip_forward")
	}
	if got := resources[0].String(); got != "Sysctl net.ipv4.ip_forward = 1" {
		t.Errorf("String() = %q, want %q", got, "Sysctl net.ipv4.ip_forward = 1")
	}
}

func TestRun_Sysctl_MissingKey(t *testing.T) {
	t.Parallel()

	run := newRun(New())
	run.Sysctl("", SysctlOpts{Value: "1"})
	if run.Err() == nil {
		t.Error("Sysctl() should set error on empty key")
	}
}

func TestRun_Sysctl_MissingValue(t *testing.T) {
	t.Parallel()

	run := newRun(New())
	run.Sysctl("net.ipv4.ip_forward", SysctlOpts{})
	if run.Err() == nil {
		t.Error("Sysctl() should set error on empty value")
	}
}
