package blueprints

import "github.com/TsekNet/converge/dsl"

// Linux declares a common Linux system baseline.
func Linux(r *dsl.Run) {
	r.File("/etc/motd", dsl.FileOpts{
		Content: "Managed by Converge\n",
		Mode:    0644,
	})

	// Kernel hardening via sysctl drop-in.
	r.File("/etc/sysctl.d/99-converge.conf", dsl.FileOpts{
		Content: "net.ipv4.ip_forward = 0\n" +
			"net.ipv4.conf.all.send_redirects = 0\n" +
			"net.ipv4.conf.default.accept_source_route = 0\n" +
			"kernel.sysrq = 0\n",
		Mode: 0644,
	})

	for _, pkg := range []string{"curl", "vim", "htop", "unzip", "jq"} {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Present})
	}

	r.Service("cron", dsl.ServiceOpts{
		State:  dsl.Running,
		Enable: true,
	})

	// Allow SSH, block outbound SMTP.
	r.Firewall("Allow SSH", dsl.FirewallOpts{Port: 22, Action: "allow"})
	r.Firewall("Block SMTP out", dsl.FirewallOpts{
		Port:      25,
		Direction: "outbound",
		Action:    "block",
	})
}
