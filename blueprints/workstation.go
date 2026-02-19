package blueprints

import "github.com/TsekNet/converge/dsl"

// Workstation declares desired state for a developer workstation.
func Workstation(r *dsl.Run) {
	r.File("/etc/motd", dsl.FileOpts{
		Content: "Managed by Converge\n",
		Mode:    0644,
	})

	r.Package("git", dsl.PackageOpts{State: dsl.Present})
	r.Package("curl", dsl.PackageOpts{State: dsl.Present})
	r.Package("neovim", dsl.PackageOpts{State: dsl.Present})

	r.Service("sshd", dsl.ServiceOpts{
		State:  dsl.Running,
		Enable: true,
	})

	r.User("devuser", dsl.UserOpts{
		Groups: []string{"sudo"},
		Shell:  "/bin/bash",
	})
}
