package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	info := Detect()
	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
	t.Logf("Detected: OS=%s Distro=%s PkgManager=%s InitSystem=%s Arch=%s",
		info.OS, info.Distro, info.PkgManager, info.InitSystem, info.Arch)
}

func TestDetect_PlatformSpecific(t *testing.T) {
	info := Detect()

	tests := []struct {
		name       string
		goos       string
		field      string
		got        string
		want       string
	}{
		{"darwin init system", "darwin", "InitSystem", info.InitSystem, "launchd"},
		{"darwin distro", "darwin", "Distro", info.Distro, "macos"},
		{"windows init system", "windows", "InitSystem", info.InitSystem, "windows"},
		{"windows distro", "windows", "Distro", info.Distro, "windows"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS != tt.goos {
				t.Skipf("skipping: requires %s", tt.goos)
			}
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.field, tt.got, tt.want)
			}
		})
	}
}

func TestDetectLinuxDistro(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	distro := detectLinuxDistro()
	if distro == "" {
		t.Error("distro should not be empty on linux")
	}
	t.Logf("Detected distro: %s", distro)
}

func TestDetectLinuxPkgManager(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	mgr := detectLinuxPkgManager()
	t.Logf("Detected pkg manager: %q", mgr)
}

func TestDetectLinuxInitSystem(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	init := detectLinuxInitSystem()
	t.Logf("Detected init system: %q", init)
}

func TestDetect_LinuxAssertions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}

	info := Detect()

	if info.OS != "linux" {
		t.Errorf("OS = %q, want %q", info.OS, "linux")
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.Distro == "" || info.Distro == "linux" {
		t.Errorf("Distro = %q, want actual distro name (e.g. ubuntu)", info.Distro)
	}
	if info.InitSystem != "systemd" && info.InitSystem != "openrc" {
		t.Errorf("InitSystem = %q, want systemd or openrc", info.InitSystem)
	}
}

func TestIsRoot(t *testing.T) {
	t.Parallel()

	// Tests never run as root, so IsRoot should return false.
	if IsRoot() {
		t.Error("IsRoot() = true, want false (tests should not run as root)")
	}
}

func TestDetectDarwinPkgManager(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	mgr := detectDarwinPkgManager()
	t.Logf("Detected darwin pkg manager: %q", mgr)
}

func TestDetectWindowsPkgManager(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	mgr := detectWindowsPkgManager()
	t.Logf("Detected windows pkg manager: %q", mgr)
}
