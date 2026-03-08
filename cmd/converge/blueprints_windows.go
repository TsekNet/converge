//go:build windows

package main

import (
	"github.com/TsekNet/converge/blueprints"
	"github.com/TsekNet/converge/blueprints/cis"
	"github.com/TsekNet/converge/dsl"
)

func registerPlatformBlueprints(a *dsl.App) {
	a.Register("windows", "Windows workstation", blueprints.Windows)
	a.Register("windows_cis", "CIS Windows 11 Enterprise L1 benchmark", cis.WindowsCIS)
}
