//go:build linux

package main

import (
	"github.com/TsekNet/converge/blueprints/cis"
	"github.com/TsekNet/converge/dsl"
)

func registerPlatformBlueprints(a *dsl.App) {
	a.Register("cis", "CIS L1 security benchmark", cis.LinuxCIS)
}
