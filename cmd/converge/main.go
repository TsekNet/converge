package main

import (
	"fmt"
	"os"

	"github.com/TsekNet/converge/blueprints"
	"github.com/TsekNet/converge/dsl"
)

var app = newApp()

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newApp() *dsl.App {
	a := dsl.New()
	a.Register("workstation", "Developer workstation baseline", blueprints.Workstation)
	a.Register("linux", "Common Linux system baseline", blueprints.Linux)
	a.Register("linux_server", "Hardened Linux server", blueprints.LinuxServer)
	a.Register("darwin", "macOS workstation", blueprints.Darwin)
	a.Register("windows", "Windows workstation", blueprints.Windows)
	a.Register("windows_cis", "CIS Windows 11 Enterprise L1 benchmark", blueprints.WindowsCIS)
	return a
}
