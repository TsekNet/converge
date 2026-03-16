package blueprints

import "github.com/TsekNet/converge/dsl"

// Undo removes everything the Baseline blueprint installs.
// For testing only: verifies converge can reverse its own changes.
func Undo(r *dsl.Run) {
	p := r.Platform()

	packages := []string{"neovim"}
	if p.PkgManager == "winget" {
		packages = []string{"Neovim.Neovim"}
	}
	for _, pkg := range packages {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Absent})
	}

	r.Firewall("Allow SSH", dsl.FirewallOpts{
		Port:   22,
		Action: "allow",
		State:  dsl.Absent,
	})
}
