package blueprints

import "github.com/TsekNet/converge/dsl"

// LinuxServer declares desired state for a hardened Linux server.
func LinuxServer(r *dsl.Run) {
	// Pull in the base Linux blueprint first, then layer server-specific hardening.
	r.Include("linux")

	// Drop-in config under sshd_config.d/ so we don't clobber the distro's defaults.
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

	r.Package("fail2ban", dsl.PackageOpts{State: dsl.Present})

	r.Service("fail2ban", dsl.ServiceOpts{
		State:  dsl.Running,
		Enable: true,
	})

	// Allow only SSH and monitoring inbound.
	r.Firewall("Allow SSH", dsl.FirewallOpts{Port: 22, Action: "allow"})
	r.Firewall("Allow monitoring", dsl.FirewallOpts{
		Port:   9090,
		Source: "10.0.0.0/8",
		Action: "allow",
	})
}
