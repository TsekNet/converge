package blueprints

import "github.com/TsekNet/converge/dsl"

// Workstation declares desired state for a developer workstation.
// Cross-platform: uses runtime detection for platform-specific paths.
func Workstation(r *dsl.Run) {
	p := r.Platform()

	// Common packages across all platforms.
	for _, pkg := range []string{"git", "curl", "neovim"} {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Present})
	}

	// Allow SSH inbound on all platforms.
	r.Firewall("Allow SSH", dsl.FirewallOpts{
		Port:   22,
		Action: "allow",
	})

	// Platform-specific MOTD.
	if p.OS == "linux" || p.OS == "darwin" {
		r.File("/etc/motd", dsl.FileOpts{
			Content: "Managed by Converge\n",
			Mode:    0644,
		})
	}

	if p.OS == "linux" {
		r.Service("sshd", dsl.ServiceOpts{
			State:  dsl.Running,
			Enable: true,
		})

		r.User("devuser", dsl.UserOpts{
			Groups: []string{"sudo"},
			Shell:  "/bin/bash",
		})
	}

	// Canary rollout: 10% of fleet gets the experimental monitoring agent.
	if r.InShard(10) {
		r.Package("converge-telemetry", dsl.PackageOpts{State: dsl.Present})
	}
}
