//go:build darwin

package cis

import "github.com/TsekNet/converge/dsl"

// DarwinCIS enforces CIS macOS 15 Sequoia v2.0.0 L1 benchmark settings.
func DarwinCIS(r *dsl.Run) {
	cisUpdates(r)
	cisFirewall(r)
	cisSharing(r)
	cisPrivacy(r)
	cisSecurity(r)
	cisScreenSaver(r)
	cisMisc(r)
}

// cisUpdates enables automatic software updates and critical patches (CIS 1.1).
func cisUpdates(r *dsl.Run) {
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "AutomaticCheckEnabled", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "AutomaticDownload", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "CriticalUpdateInstall", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "AutomaticallyInstallMacOSUpdates", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.commerce", dsl.PlistOpts{Key: "AutoUpdate", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "ConfigDataInstall", Value: true, Type: "bool", Host: true})
}

// cisFirewall enables the application-layer firewall and stealth mode (CIS 2.2).
// Stealth mode prevents the Mac from responding to probing requests (ICMP/port scans).
func cisFirewall(r *dsl.Run) {
	r.Exec("enable-firewall", dsl.ExecOpts{
		Command: "/usr/libexec/ApplicationFirewall/socketfilterfw",
		Args:    []string{"--setglobalstate", "on"},
		OnlyIf:  "/usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate | grep -q enabled",
	})
	r.Exec("enable-stealth", dsl.ExecOpts{
		Command: "/usr/libexec/ApplicationFirewall/socketfilterfw",
		Args:    []string{"--setstealthmode", "on"},
		OnlyIf:  "/usr/libexec/ApplicationFirewall/socketfilterfw --getstealthmode | grep -q enabled",
	})
}

// cisSharing disables sharing services that expose the Mac to the network (CIS 2.3.x).
func cisSharing(r *dsl.Run) {
	r.Exec("disable-screen-sharing", dsl.ExecOpts{
		Command: "launchctl",
		Args:    []string{"disable", "system/com.apple.screensharing"},
	})
	r.Exec("disable-file-sharing", dsl.ExecOpts{
		Command: "launchctl",
		Args:    []string{"disable", "system/com.apple.smbd"},
	})
	r.Plist("com.apple.mcxprinting", dsl.PlistOpts{Key: "PrinterSharing", Value: false, Type: "bool", Host: true})
	r.Exec("disable-remote-login", dsl.ExecOpts{
		Command: "launchctl",
		Args:    []string{"disable", "system/com.apple.sshd"},
		OnlyIf:  "launchctl print system/com.apple.sshd 2>/dev/null | grep -q 'state = disabled'",
	})
	r.Exec("disable-remote-management", dsl.ExecOpts{
		Command: "launchctl",
		Args:    []string{"disable", "system/com.apple.remotemanagement"},
		OnlyIf:  "launchctl print system/com.apple.remotemanagement 2>/dev/null | grep -q 'state = disabled'",
	})
	r.Exec("disable-remote-apple-events", dsl.ExecOpts{
		Command: "launchctl",
		Args:    []string{"disable", "system/com.apple.AEServer"},
	})
	r.Plist("com.apple.MCX", dsl.PlistOpts{Key: "forceInternetSharingOff", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.Bluetooth", dsl.PlistOpts{Key: "PrefKeyServicesEnabled", Value: false, Type: "bool", Host: true})
}

// cisPrivacy disables Siri, Apple Intelligence, analytics, and ad tracking (CIS 2.5.x).
func cisPrivacy(r *dsl.Run) {
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "Assistant Enabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.Siri", dsl.PlistOpts{Key: "VoiceTriggerUserEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "ExternalIntelligenceEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "WritingToolsEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.SubmitDiagInfo", dsl.PlistOpts{Key: "AutoSubmit", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "Siri Data Sharing Opt-In Status", Value: 2, Type: "int", Host: true}) // 2 = opted out
	r.Plist("com.apple.SubmitDiagInfo", dsl.PlistOpts{Key: "ThirdPartyDataSubmit", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.SubmitDiagInfo", dsl.PlistOpts{Key: "iCloudAnalyticsSubmit", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.AdLib", dsl.PlistOpts{Key: "allowApplePersonalizedAdvertising", Value: false, Type: "bool", Host: true})
}

// cisSecurity enforces Gatekeeper, FileVault, and guest account restrictions (CIS 2.6.x).
func cisSecurity(r *dsl.Run) {
	r.Exec("enable-gatekeeper", dsl.ExecOpts{
		Command: "spctl",
		Args:    []string{"--master-enable"},
		OnlyIf:  "spctl --status 2>&1 | grep -q enabled",
	})
	r.Exec("check-filevault", dsl.ExecOpts{
		Command: "fdesetup",
		Args:    []string{"status"},
		OnlyIf:  "fdesetup status | grep -q 'FileVault is On'",
	})
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "GuestEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.AppleFileServer", dsl.PlistOpts{Key: "guestAccess", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.smb.server", dsl.PlistOpts{Key: "AllowGuestAccess", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "RetriesUntilHint", Value: 0, Type: "int", Host: true})
}

// cisScreenSaver sets inactivity timeout and login window behavior (CIS 2.10.x).
func cisScreenSaver(r *dsl.Run) {
	r.Plist("com.apple.screensaver", dsl.PlistOpts{Key: "idleTime", Value: 900, Type: "int", Host: true}) // 900 seconds = 15 minutes
	r.Plist("com.apple.screensaver", dsl.PlistOpts{Key: "askForPassword", Value: 1, Type: "int", Host: true})
	r.Plist("com.apple.screensaver", dsl.PlistOpts{Key: "askForPasswordDelay", Value: 5, Type: "int", Host: true}) // 5 seconds grace period
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "SHOWFULLNAME", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "LoginwindowText", Value: "Authorized uses only. All activity may be monitored and reported.", Type: "string", Host: true})
}

// cisMisc covers NTP, power management, AirDrop, and AirPlay settings (CIS 2.x).
func cisMisc(r *dsl.Run) {
	r.Plist("com.apple.timed", dsl.PlistOpts{Key: "TMAutomaticTimeOnlyEnabled", Value: true, Type: "bool", Host: true})
	r.Exec("disable-powernap", dsl.ExecOpts{
		Command: "pmset",
		Args:    []string{"-a", "powernap", "0"},
		OnlyIf:  "pmset -g | grep -q 'powernap.*0'",
	})
	r.Exec("disable-wake-network", dsl.ExecOpts{
		Command: "pmset",
		Args:    []string{"-a", "womp", "0"},
		OnlyIf:  "pmset -g | grep -q 'womp.*0'",
	})
	r.Plist("com.apple.NetworkBrowser", dsl.PlistOpts{Key: "DisableAirDrop", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.controlcenter", dsl.PlistOpts{Key: "AirplayRecieverEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.lookup.shared", dsl.PlistOpts{Key: "LookupSuggestionsDisabled", Value: true, Type: "bool", Host: true})
}
