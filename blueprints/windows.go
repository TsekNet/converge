package blueprints

import "github.com/TsekNet/converge/dsl"

// Windows declares desired state for a Windows workstation.
func Windows(r *dsl.Run) {
	for _, pkg := range []string{"git", "7zip", "vscode", "curl"} {
		r.Package(pkg, dsl.PackageOpts{State: dsl.Present})
	}

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
}
