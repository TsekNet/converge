package blueprints

import "github.com/TsekNet/converge/dsl"

// LinuxServer declares desired state for a hardened Linux server.
func LinuxServer(r *dsl.Run) {
	r.Include("linux")

	r.File("/etc/ssh/sshd_config.d/converge.conf", dsl.FileOpts{
		Content: "PermitRootLogin no\n" +
			"PasswordAuthentication no\n" +
			"X11Forwarding no\n" +
			"MaxAuthTries 3\n",
		Mode:     0600,
		Critical: true,
	})

	r.Service("sshd", dsl.ServiceOpts{
		State:    dsl.Running,
		Enable:   true,
		Critical: true,
	})

	for _, pkg := range []string{"fail2ban", "ufw"} {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Present})
	}

	r.Service("fail2ban", dsl.ServiceOpts{
		State:  dsl.Running,
		Enable: true,
	})
}
