//go:build windows

package blueprints

import "github.com/TsekNet/converge/dsl"

// Windows declares desired state for Windows.
func Windows(r *dsl.Run) {
	for _, pkg := range []string{"git", "7zip", "vscode", "curl"} {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Present})
	}

	// Show file extensions and disable telemetry via registry.
	r.Registry(`HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, dsl.RegistryOpts{
		Value: "HideFileExt",
		Type:  "dword",
		Data:  0,
	})

	r.Registry(`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, dsl.RegistryOpts{
		Value: "AllowTelemetry",
		Type:  "dword",
		Data:  0,
	})

	// Allow RDP inbound, block outbound SMTP.
	r.Firewall("Allow RDP", dsl.FirewallOpts{Port: 3389, Action: "allow"})
	r.Firewall("Block SMTP out", dsl.FirewallOpts{
		Port:      25,
		Direction: "outbound",
		Action:    "block",
	})

	// Canary: 5% of fleet gets the new security agent.
	if r.InShard(5) {
		r.Package("converge-defender", dsl.PackageOpts{State: dsl.Present})
	}
}
