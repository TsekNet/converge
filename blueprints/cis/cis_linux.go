//go:build linux

package cis

import (
	"os"

	"github.com/TsekNet/converge/dsl"
)

// LinuxCIS enforces CIS Ubuntu Linux 24.04 LTS L1 Server benchmark settings.
func LinuxCIS(r *dsl.Run) {
	cisFilesystems(r)
	cisSysctl(r)
	cisServices(r)
	cisPackages(r)
	cisSSH(r)
	cisPAM(r)
	cisBanners(r)
	cisPermissions(r)
	cisAudit(r)
	cisAuth(r)
}

// cisFilesystems blacklists kernel modules for uncommon/legacy filesystems (CIS 1.1.x).
func cisFilesystems(r *dsl.Run) {
	// Prevent mounting of these filesystem types by redirecting their install to /bin/false.
	modules := []string{
		"cramfs", "freevxfs", "hfs", "hfsplus", "jffs2", "usb-storage",
	}
	for _, m := range modules {
		r.File("/etc/modprobe.d/cis-"+m+".conf", dsl.FileOpts{
			Content: "install " + m + " /bin/false\nblacklist " + m + "\n",
			Mode:    0644,
		})
	}
}

// cisSysctl hardens kernel networking and memory protections via /proc/sys (CIS 3.x).
func cisSysctl(r *dsl.Run) {
	type param struct {
		key, value string
	}
	params := []param{
		{"kernel.randomize_va_space", "2"},           // ASLR: full randomization
		{"kernel.yama.ptrace_scope", "1"},            // restrict ptrace to parent processes
		{"net.ipv4.ip_forward", "0"},
		{"net.ipv4.conf.all.send_redirects", "0"},
		{"net.ipv4.conf.default.send_redirects", "0"},
		{"net.ipv4.conf.all.accept_source_route", "0"},
		{"net.ipv4.conf.default.accept_source_route", "0"},
		{"net.ipv4.conf.all.accept_redirects", "0"},
		{"net.ipv4.conf.default.accept_redirects", "0"},
		{"net.ipv4.conf.all.secure_redirects", "0"},
		{"net.ipv4.conf.default.secure_redirects", "0"},
		{"net.ipv4.conf.all.log_martians", "1"},
		{"net.ipv4.conf.default.log_martians", "1"},
		{"net.ipv4.icmp_echo_ignore_broadcasts", "1"},
		{"net.ipv4.icmp_ignore_bogus_error_responses", "1"},
		{"net.ipv4.conf.all.rp_filter", "1"},
		{"net.ipv4.conf.default.rp_filter", "1"},
		{"net.ipv4.tcp_syncookies", "1"},              // SYN flood protection
		{"net.ipv6.conf.all.accept_ra", "0"},
		{"net.ipv6.conf.default.accept_ra", "0"},
		{"net.ipv6.conf.all.accept_redirects", "0"},
		{"net.ipv6.conf.default.accept_redirects", "0"},
		{"net.ipv6.conf.all.accept_source_route", "0"},
		{"net.ipv6.conf.default.accept_source_route", "0"},
		{"net.ipv4.conf.all.forwarding", "0"},
		{"net.ipv6.conf.all.forwarding", "0"},
		{"fs.suid_dumpable", "0"},                     // prevent core dumps from SUID binaries
		{"kernel.core_uses_pid", "1"},
	}
	for _, p := range params {
		r.Sysctl(p.key, dsl.SysctlOpts{Value: p.value, Persist: true})
	}
}

// cisServices disables network-facing services not needed on a hardened server (CIS 2.x).
func cisServices(r *dsl.Run) {
	services := []string{
		"avahi-daemon", "cups", "isc-dhcp-server", "slapd",
		"nfs-server", "bind9", "vsftpd", "apache2",
		"dovecot", "smbd", "squid", "snmpd",
		"rpcbind", "rsync",
	}
	for _, name := range services {
		r.Service(name, dsl.ServiceOpts{State: dsl.Stopped, Enable: false})
	}
}

// cisPackages removes legacy/insecure clients and ensures security tooling is present (CIS 2.x).
func cisPackages(r *dsl.Run) {
	remove := []string{
		"telnet", "nis", "rsh-client", "talk",
		"ldap-utils", "rpcbind", "xinetd",
	}
	for _, name := range remove {
		r.Package(name, dsl.PackageOpts{State: dsl.Absent})
	}

	ensure := []string{"apparmor", "auditd", "aide", "libpam-pwquality"}
	for _, name := range ensure {
		r.Package(name, dsl.PackageOpts{State: dsl.Present})
	}
}

// cisSSH writes a hardened sshd_config aligned with CIS 5.2.x recommendations.
func cisSSH(r *dsl.Run) {
	r.File("/etc/ssh/sshd_config", dsl.FileOpts{
		Mode: 0600,
		Content: `Protocol 2
PermitRootLogin no
MaxAuthTries 4
MaxSessions 10
PubkeyAuthentication yes
HostbasedAuthentication no
PermitEmptyPasswords no
PermitUserEnvironment no
IgnoreRhosts yes
X11Forwarding no
AllowTcpForwarding no
AllowAgentForwarding no
Banner /etc/issue.net
UsePAM yes
ClientAliveInterval 300
ClientAliveCountMax 3
LoginGraceTime 60
MaxStartups 10:30:60
LogLevel VERBOSE
Ciphers aes256-gcm@openssh.com,aes128-gcm@openssh.com,aes256-ctr,aes192-ctr,aes128-ctr
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512,hmac-sha2-256
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp521,ecdh-sha2-nistp384,ecdh-sha2-nistp256,diffie-hellman-group-exchange-sha256
`,
	})
}

// cisPAM configures password complexity requirements via pam_pwquality (CIS 5.3.x).
func cisPAM(r *dsl.Run) {
	r.File("/etc/security/pwquality.conf", dsl.FileOpts{
		Mode: 0644,
		Content: `minlen = 14
dcredit = -1
ucredit = -1
ocredit = -1
lcredit = -1
minclass = 4
maxrepeat = 3
maxclassrepeat = 0
gecoscheck = 1
`,
	})
}

// cisBanners sets legal/warning banners shown at login (CIS 1.7.x).
func cisBanners(r *dsl.Run) {
	banner := "Authorized uses only. All activity may be monitored and reported.\n"
	for _, path := range []string{"/etc/motd", "/etc/issue", "/etc/issue.net"} {
		r.File(path, dsl.FileOpts{Content: banner, Mode: 0644})
	}
}

// cisPermissions locks down ownership and modes on sensitive system files (CIS 6.1.x).
func cisPermissions(r *dsl.Run) {
	type perm struct {
		path, owner, group string
		mode               uint32
	}
	perms := []perm{
		{"/etc/passwd", "root", "root", 0644},
		{"/etc/passwd-", "root", "root", 0600},
		{"/etc/shadow", "root", "shadow", 0640},
		{"/etc/shadow-", "root", "root", 0600},
		{"/etc/group", "root", "root", 0644},
		{"/etc/group-", "root", "root", 0600},
		{"/etc/gshadow", "root", "shadow", 0640},
		{"/etc/gshadow-", "root", "root", 0600},
		{"/etc/crontab", "root", "root", 0600},
		{"/etc/cron.d", "root", "root", 0700},
		{"/etc/cron.daily", "root", "root", 0700},
		{"/etc/cron.hourly", "root", "root", 0700},
		{"/etc/cron.weekly", "root", "root", 0700},
		{"/etc/cron.monthly", "root", "root", 0700},
	}
	for _, p := range perms {
		r.File(p.path, dsl.FileOpts{
			Mode:  os.FileMode(p.mode),
			Owner: p.owner,
			Group: p.group,
		})
	}
}

// cisAudit enables auditd and deploys comprehensive audit rules (CIS 4.1.x).
func cisAudit(r *dsl.Run) {
	r.Service("auditd", dsl.ServiceOpts{State: dsl.Running, Enable: true})

	// Rules cover: time changes, identity files, network config, MAC policy,
	// logins, sessions, permission changes, unauthorized access, mounts, deletes, sudo, kernel modules.
	// Ends with "-e 2" to lock audit config (requires reboot to change rules).
	r.File("/etc/audit/rules.d/cis.rules", dsl.FileOpts{
		Mode: 0640,
		Content: `-a always,exit -F arch=b64 -S adjtimex -S settimeofday -k time-change
-a always,exit -F arch=b32 -S adjtimex -S settimeofday -S stime -k time-change
-a always,exit -F arch=b64 -S clock_settime -k time-change
-a always,exit -F arch=b32 -S clock_settime -k time-change
-w /etc/localtime -p wa -k time-change

-w /etc/group -p wa -k identity
-w /etc/passwd -p wa -k identity
-w /etc/gshadow -p wa -k identity
-w /etc/shadow -p wa -k identity
-w /etc/security/opasswd -p wa -k identity

-a always,exit -F arch=b64 -S sethostname -S setdomainname -k system-locale
-a always,exit -F arch=b32 -S sethostname -S setdomainname -k system-locale
-w /etc/issue -p wa -k system-locale
-w /etc/issue.net -p wa -k system-locale
-w /etc/hosts -p wa -k system-locale
-w /etc/network -p wa -k system-locale

-w /etc/apparmor/ -p wa -k MAC-policy
-w /etc/apparmor.d/ -p wa -k MAC-policy

-w /var/log/faillog -p wa -k logins
-w /var/log/lastlog -p wa -k logins
-w /var/log/tallylog -p wa -k logins

-w /var/run/utmp -p wa -k session
-w /var/log/wtmp -p wa -k session
-w /var/log/btmp -p wa -k session

-a always,exit -F arch=b64 -S chmod -S fchmod -S fchmodat -k perm_mod
-a always,exit -F arch=b32 -S chmod -S fchmod -S fchmodat -k perm_mod
-a always,exit -F arch=b64 -S chown -S fchown -S fchownat -S lchown -k perm_mod
-a always,exit -F arch=b32 -S chown -S fchown -S fchownat -S lchown -k perm_mod
-a always,exit -F arch=b64 -S setxattr -S lsetxattr -S fsetxattr -S removexattr -S lremovexattr -S fremovexattr -k perm_mod
-a always,exit -F arch=b32 -S setxattr -S lsetxattr -S fsetxattr -S removexattr -S lremovexattr -S fremovexattr -k perm_mod

-a always,exit -F arch=b64 -S creat -S open -S openat -S truncate -S ftruncate -F exit=-EACCES -k access
-a always,exit -F arch=b32 -S creat -S open -S openat -S truncate -S ftruncate -F exit=-EACCES -k access
-a always,exit -F arch=b64 -S creat -S open -S openat -S truncate -S ftruncate -F exit=-EPERM -k access
-a always,exit -F arch=b32 -S creat -S open -S openat -S truncate -S ftruncate -F exit=-EPERM -k access

-a always,exit -F arch=b64 -S mount -k mounts
-a always,exit -F arch=b32 -S mount -k mounts

-a always,exit -F arch=b64 -S unlink -S unlinkat -S rename -S renameat -k delete
-a always,exit -F arch=b32 -S unlink -S unlinkat -S rename -S renameat -k delete

-w /etc/sudoers -p wa -k scope
-w /etc/sudoers.d/ -p wa -k scope

-w /var/log/sudo.log -p wa -k actions

-w /sbin/insmod -p x -k modules
-w /sbin/rmmod -p x -k modules
-w /sbin/modprobe -p x -k modules
-a always,exit -F arch=b64 -S init_module -S delete_module -k modules

-e 2
`,
	})
}

// cisAuth configures password aging and login defaults (CIS 5.4.x).
func cisAuth(r *dsl.Run) {
	r.File("/etc/login.defs", dsl.FileOpts{
		Mode: 0644,
		Content: `PASS_MAX_DAYS 365
PASS_MIN_DAYS 1
PASS_WARN_AGE 7
UMASK 027
LOGIN_RETRIES 5
LOGIN_TIMEOUT 60
ENCRYPT_METHOD SHA512
`,
	})
}
