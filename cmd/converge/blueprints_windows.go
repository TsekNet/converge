//go:build windows

package main

import (
	"github.com/TsekNet/converge/blueprints"
	"github.com/TsekNet/converge/blueprints/cis"
	"github.com/TsekNet/converge/dsl"
)

func registerPlatformBlueprints(a *dsl.App) {
	a.Register("windows", "Windows workstation", blueprints.Windows)
	a.Register("cis", "CIS L1 security benchmark", cis.WindowsCIS)
}
