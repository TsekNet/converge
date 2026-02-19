package pkg

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type mockManager struct {
	name      string
	installed map[string]bool
	installFn func(string) error
	removeFn  func(string) error
}

func (m *mockManager) Name() string { return m.name }
func (m *mockManager) IsInstalled(_ context.Context, name string) (bool, error) {
	return m.installed[name], nil
}
func (m *mockManager) Install(_ context.Context, name string) error {
	if m.installFn != nil {
		return m.installFn(name)
	}
	m.installed[name] = true
	return nil
}
func (m *mockManager) Remove(_ context.Context, name string) error {
	if m.removeFn != nil {
		return m.removeFn(name)
	}
	delete(m.installed, name)
	return nil
}

func TestPackage_Check(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		pkgName   string
		state     string
		installed bool
		wantSync  bool
	}{
		{"present and installed", "git", "present", true, true},
		{"present but missing", "git", "present", false, false},
		{"absent and missing", "git", "absent", false, true},
		{"absent but installed", "git", "absent", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &mockManager{name: "mock", installed: map[string]bool{tt.pkgName: tt.installed}}
			p := &Package{PkgName: tt.pkgName, State: tt.state, Manager: mgr}

			state, err := p.Check(ctx)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			if state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", state.InSync, tt.wantSync)
			}
		})
	}
}

func TestPackage_Apply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		state   string
		wantMsg string
	}{
		{"install", "present", "installed"},
		{"remove", "absent", "removed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &mockManager{name: "mock", installed: map[string]bool{}}
			p := &Package{PkgName: "git", State: tt.state, Manager: mgr}

			result, err := p.Apply(ctx)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if result.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.wantMsg)
			}
		})
	}
}

func TestPackage_NoManager(t *testing.T) {
	ctx := context.Background()

	p := &Package{PkgName: "git", State: "present", Manager: nil, ManagerName: "zypper"}

	_, err := p.Check(ctx)
	if err == nil {
		t.Error("Check() should fail with no manager")
	}

	_, err = p.Apply(ctx)
	if err == nil {
		t.Error("Apply() should fail with no manager")
	}
}

func TestPackage_IDAndString(t *testing.T) {
	tests := []struct {
		pkgName string
		wantID  string
		wantStr string
	}{
		{"git", "package:git", "Package git"},
		{"neovim", "package:neovim", "Package neovim"},
	}
	for _, tt := range tests {
		t.Run(tt.pkgName, func(t *testing.T) {
			p := &Package{PkgName: tt.pkgName}
			if p.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", p.ID(), tt.wantID)
			}
			if p.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", p.String(), tt.wantStr)
			}
		})
	}
}

func TestDetectManager(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{"apt", false},
		{"brew", false},
		{"choco", false},
		{"dnf", false},
		{"yum", false},
		{"zypper", false},
		{"apk", false},
		{"pacman", false},
		{"winget", false},
		{"unknown", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := detectManager(tt.name)
			if (mgr == nil) != tt.wantNil {
				t.Errorf("detectManager(%q) nil = %v, want %v", tt.name, mgr == nil, tt.wantNil)
			}
		})
	}
}

func TestManagerNames(t *testing.T) {
	tests := []struct {
		managerName string
		wantName    string
	}{
		{"apt", "apt"},
		{"brew", "brew"},
		{"choco", "choco"},
		{"dnf", "dnf"},
		{"yum", "yum"},
		{"zypper", "zypper"},
		{"apk", "apk"},
		{"pacman", "pacman"},
		{"winget", "winget"},
	}
	for _, tt := range tests {
		t.Run(tt.managerName, func(t *testing.T) {
			mgr := detectManager(tt.managerName)
			if mgr == nil {
				t.Fatalf("detectManager(%q) returned nil", tt.managerName)
			}
			if mgr.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", mgr.Name(), tt.wantName)
			}
		})
	}
}

func TestPackage_Check_PresentNotInstalled(t *testing.T) {
	ctx := context.Background()
	mgr := &mockManager{name: "mock", installed: map[string]bool{}}
	p := &Package{PkgName: "vim", State: "present", Manager: mgr}

	state, err := p.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if state.InSync {
		t.Error("should not be in sync")
	}
	if len(state.Changes) != 1 {
		t.Fatalf("want 1 change, got %d", len(state.Changes))
	}
	if state.Changes[0].Action != "add" {
		t.Errorf("action = %q, want %q", state.Changes[0].Action, "add")
	}
	if !strings.Contains(state.Changes[0].To, "install via") {
		t.Errorf("To = %q, should contain 'install via'", state.Changes[0].To)
	}
}

func TestAptManager_IsInstalled_Live(t *testing.T) {
	ctx := context.Background()
	mgr := &aptManager{}

	tests := []struct {
		pkg       string
		wantFound bool
	}{
		{"coreutils", true},
		{"definitely-not-a-real-package-xyz", false},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			found, err := mgr.IsInstalled(ctx, tt.pkg)
			if err != nil {
				t.Fatalf("IsInstalled() error = %v", err)
			}
			if found != tt.wantFound {
				t.Errorf("IsInstalled(%q) = %v, want %v", tt.pkg, found, tt.wantFound)
			}
		})
	}
}

func TestPackage_IsCritical(t *testing.T) {
	p := &Package{PkgName: "git", Critical: false}
	if p.IsCritical() {
		t.Error("IsCritical() = true, want false")
	}
	p.Critical = true
	if !p.IsCritical() {
		t.Error("IsCritical() = false, want true")
	}
}

func TestNew(t *testing.T) {
	p := New("curl", "present", "apt")
	if p.PkgName != "curl" {
		t.Errorf("PkgName = %q, want %q", p.PkgName, "curl")
	}
	if p.State != "present" {
		t.Errorf("State = %q, want %q", p.State, "present")
	}
	if p.ManagerName != "apt" {
		t.Errorf("ManagerName = %q, want %q", p.ManagerName, "apt")
	}
	if p.Manager == nil {
		t.Error("Manager should not be nil for apt")
	}

	p2 := New("curl", "absent", "nonexistent")
	if p2.Manager != nil {
		t.Error("Manager should be nil for unknown manager")
	}
}

func TestPackage_Check_AbsentChanges(t *testing.T) {
	ctx := context.Background()
	mgr := &mockManager{name: "mock", installed: map[string]bool{"git": true}}
	p := &Package{PkgName: "git", State: "absent", Manager: mgr}

	state, err := p.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if state.InSync {
		t.Error("should not be in sync when installed but want absent")
	}
	if len(state.Changes) != 1 {
		t.Fatalf("want 1 change, got %d", len(state.Changes))
	}
	if state.Changes[0].Action != "remove" {
		t.Errorf("action = %q, want %q", state.Changes[0].Action, "remove")
	}
}

func TestAllManagers_IsInstalled_Graceful(t *testing.T) {
	ctx := context.Background()
	managers := []string{"apt", "brew", "choco", "dnf", "yum", "zypper", "apk", "pacman", "winget"}
	for _, name := range managers {
		t.Run(name, func(t *testing.T) {
			mgr := detectManager(name)
			if mgr == nil {
				t.Fatalf("detectManager(%q) = nil", name)
			}
			installed, err := mgr.IsInstalled(ctx, "converge-nonexistent-pkg-xyz")
			if err != nil {
				t.Logf("IsInstalled returned error (expected if %s not on PATH): %v", name, err)
			}
			if installed {
				t.Errorf("nonexistent package should not be installed via %s", name)
			}
		})
	}
}

func TestAllManagers_Install_Graceful(t *testing.T) {
	ctx := context.Background()
	managers := []string{"dnf", "yum", "zypper", "apk", "pacman", "winget", "brew", "choco"}
	for _, name := range managers {
		t.Run(name, func(t *testing.T) {
			mgr := detectManager(name)
			if mgr == nil {
				t.Fatalf("detectManager(%q) = nil", name)
			}
			err := mgr.Install(ctx, "converge-nonexistent-pkg-xyz")
			if err == nil {
				t.Logf("Install unexpectedly succeeded for %s (may be installed)", name)
			}
		})
	}
}

func TestAllManagers_Remove_Graceful(t *testing.T) {
	ctx := context.Background()
	managers := []string{"dnf", "yum", "zypper", "apk", "pacman", "winget", "brew", "choco"}
	for _, name := range managers {
		t.Run(name, func(t *testing.T) {
			mgr := detectManager(name)
			if mgr == nil {
				t.Fatalf("detectManager(%q) = nil", name)
			}
			err := mgr.Remove(ctx, "converge-nonexistent-pkg-xyz")
			if err == nil {
				t.Logf("Remove unexpectedly succeeded for %s", name)
			}
		})
	}
}

func TestPackage_Apply_Error(t *testing.T) {
	ctx := context.Background()
	mgr := &mockManager{
		name:      "mock",
		installed: map[string]bool{},
		installFn: func(string) error { return fmt.Errorf("network error") },
	}
	p := &Package{PkgName: "git", State: "present", Manager: mgr}

	_, err := p.Apply(ctx)
	if err == nil {
		t.Error("Apply() should return error")
	}
}
