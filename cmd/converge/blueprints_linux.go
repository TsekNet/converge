//go:build linux

package main

import (
	"github.com/TsekNet/converge/blueprints/cis"
	"github.com/TsekNet/converge/dsl"
)

func registerPlatformBlueprints(a *dsl.App) {
	a.Register("linux_cis", "CIS Ubuntu Linux 24.04 LTS L1 benchmark", cis.LinuxCIS)
}
