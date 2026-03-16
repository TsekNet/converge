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
	a.Register("baseline", "Cross-platform baseline for all managed hosts", blueprints.Baseline)
	a.Register("linux", "Common Linux system baseline", blueprints.Linux)
	a.Register("linux_server", "Hardened Linux server", blueprints.LinuxServer)
	a.Register("darwin", "macOS configuration", blueprints.Darwin)
	registerPlatformBlueprints(a)
	return a
}
