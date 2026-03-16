package blueprints

import "github.com/TsekNet/converge/dsl"

// Darwin declares desired state for macOS.
func Darwin(r *dsl.Run) {
	for _, pkg := range []string{"git", "jq", "htop", "wget", "tree"} {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Present})
	}

	r.File("/etc/motd", dsl.FileOpts{
		Content: "Managed by Converge\n",
		Mode:    0644,
	})

	// Allow SSH inbound.
	r.Firewall("Allow SSH", dsl.FirewallOpts{Port: 22, Action: "allow"})
}
