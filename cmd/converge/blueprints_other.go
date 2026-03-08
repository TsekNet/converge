//go:build !windows && !linux && !darwin

package main

import "github.com/TsekNet/converge/dsl"

func registerPlatformBlueprints(_ *dsl.App) {}
