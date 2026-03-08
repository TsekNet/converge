//go:build darwin

package main

import (
	"github.com/TsekNet/converge/blueprints/cis"
	"github.com/TsekNet/converge/dsl"
)

func registerPlatformBlueprints(a *dsl.App) {
	a.Register("darwin_cis", "CIS macOS 15 Sequoia L1 benchmark", cis.DarwinCIS)
}
