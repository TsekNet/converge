package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Info struct {
	OS         string // "linux", "darwin", "windows"
	Distro     string // "ubuntu", "fedora", "macos", "windows"
	PkgManager string // "apt", "dnf", "brew", "choco", ""
	InitSystem string // "systemd", "launchd", "windows", ""
	Arch       string // "amd64", "arm64"
}

func Detect() Info {
	info := Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	switch info.OS {
	case "linux":
		info.Distro = detectLinuxDistro()
		info.PkgManager = detectLinuxPkgManager()
		info.InitSystem = detectLinuxInitSystem()
	case "darwin":
		info.Distro = "macos"
		info.PkgManager = detectDarwinPkgManager()
		info.InitSystem = "launchd"
	case "windows":
		info.Distro = "windows"
		info.PkgManager = detectWindowsPkgManager()
		info.InitSystem = "windows"
	}

	return info
}

func detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
	}
	return "linux"
}

func detectLinuxPkgManager() string {
	managers := []string{"apt-get", "dnf", "yum", "zypper", "apk", "pacman"}
	for _, m := range managers {
		if _, err := exec.LookPath(m); err == nil {
			if m == "apt-get" {
				return "apt"
			}
			return m
		}
	}
	return ""
}

func detectLinuxInitSystem() string {
	if _, err := exec.LookPath("systemctl"); err == nil {
		return "systemd"
	}
	if _, err := os.Stat("/sbin/openrc"); err == nil {
		return "openrc"
	}
	return ""
}

func detectDarwinPkgManager() string {
	if _, err := exec.LookPath("brew"); err == nil {
		return "brew"
	}
	return ""
}

func detectWindowsPkgManager() string {
	if _, err := exec.LookPath("choco"); err == nil {
		return "choco"
	}
	if _, err := exec.LookPath("winget"); err == nil {
		return "winget"
	}
	return ""
}
