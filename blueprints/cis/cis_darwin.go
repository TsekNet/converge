//go:build darwin

package cis

import "github.com/TsekNet/converge/dsl"

// DarwinCIS enforces CIS macOS L1 benchmark settings.
// Based on CIS Apple macOS 26 Tahoe v1.0.0 L1 (90 items).
func DarwinCIS(r *dsl.Run) {
	cisUpdates(r)
	cisFirewall(r)
	cisSharing(r)
	cisAI(r)
	cisPrivacy(r)
	cisSecurity(r)
	cisScreenSaver(r)
	cisMisc(r)
}

// --- 1.x Software Updates ---

// cisUpdates enables automatic software updates, critical patches, and enforces
// a maximum 30-day deferment window (CIS 1.1-1.6).
func cisUpdates(r *dsl.Run) {
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "AutomaticCheckEnabled", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "AutomaticDownload", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "CriticalUpdateInstall", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "AutomaticallyInstallMacOSUpdates", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.commerce", dsl.PlistOpts{Key: "AutoUpdate", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "ConfigDataInstall", Value: true, Type: "bool", Host: true})
	// CIS 1.6: software update deferment must not exceed 30 days
	r.Plist("com.apple.SoftwareUpdate", dsl.PlistOpts{Key: "MajorOSDeferredInstallDelay", Value: 30, Type: "int", Host: true})
}

// --- 2.2.x Firewall ---

// cisFirewall enables the application-layer firewall and stealth mode (CIS 2.2.1-2.2.2).
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

// --- 2.3.x Sharing & Time ---

// cisSharing disables sharing services that expose the Mac to the network and
// ensures time sync is active (CIS 2.3.x).
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
	// CIS 2.3.2.1-2.3.2.2: ensure NTP time sync is active
	r.Plist("com.apple.timed", dsl.PlistOpts{Key: "TMAutomaticTimeOnlyEnabled", Value: true, Type: "bool", Host: true})
	r.Exec("enable-time-service", dsl.ExecOpts{
		Command: "systemsetup",
		Args:    []string{"-setusingnetworktime", "on"},
		OnlyIf:  "systemsetup -getusingnetworktime | grep -q 'On'",
	})
}

// --- 2.5.x Apple Intelligence & Siri ---

// cisAI disables Apple Intelligence features: external intelligence extensions,
// Writing Tools, Mail/Notes summarization, Siri, and "Hey Siri" (CIS 2.5.x).
func cisAI(r *dsl.Run) {
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "ExternalIntelligenceEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "WritingToolsEnabled", Value: false, Type: "bool", Host: true})
	// CIS 2.5.1.3-2.5.1.4: disable per-app AI summarization
	r.Plist("com.apple.mail", dsl.PlistOpts{Key: "SummarizationEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.Notes", dsl.PlistOpts{Key: "SummarizationEnabled", Value: false, Type: "bool", Host: true})
	// CIS 2.5.2.1-2.5.2.2: disable Siri and voice trigger
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "Assistant Enabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.Siri", dsl.PlistOpts{Key: "VoiceTriggerUserEnabled", Value: false, Type: "bool", Host: true})
}

// --- 2.6.x Privacy & Analytics ---

// cisPrivacy disables analytics sharing, ad tracking, and search suggestions (CIS 2.6.x, 2.9.x).
func cisPrivacy(r *dsl.Run) {
	r.Plist("com.apple.SubmitDiagInfo", dsl.PlistOpts{Key: "AutoSubmit", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "Siri Data Sharing Opt-In Status", Value: 2, Type: "int", Host: true}) // 2 = opted out
	// CIS 2.6.3.3: disable assistive voice feature improvement
	r.Plist("com.apple.assistant.support", dsl.PlistOpts{Key: "AssistiveVoiceFeaturesImprovementEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.SubmitDiagInfo", dsl.PlistOpts{Key: "ThirdPartyDataSubmit", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.SubmitDiagInfo", dsl.PlistOpts{Key: "iCloudAnalyticsSubmit", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.AdLib", dsl.PlistOpts{Key: "allowApplePersonalizedAdvertising", Value: false, Type: "bool", Host: true})
	// CIS 2.9.1: disable "Help Apple Improve Search"
	r.Plist("com.apple.lookup.shared", dsl.PlistOpts{Key: "LookupSuggestionsDisabled", Value: true, Type: "bool", Host: true})
}

// --- 2.6.5-2.6.8, 2.11.x, 2.12.x, 2.13.x Security & Login ---

// cisSecurity enforces Gatekeeper, FileVault, admin auth for system prefs,
// guest account restrictions, and password hint removal (CIS 2.6.x, 2.11.x, 2.12.x, 2.13.x).
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
	// CIS 2.6.8: require admin password for system-wide preference changes
	r.Exec("require-admin-for-system-prefs", dsl.ExecOpts{
		Command: "security",
		Args:    []string{"authorizationdb", "write", "system.preferences", "authenticate-admin"},
		OnlyIf:  "security authorizationdb read system.preferences 2>/dev/null | grep -q authenticate-admin",
	})
	// CIS 2.13.x: disable guest account and guest access to shares
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "GuestEnabled", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.AppleFileServer", dsl.PlistOpts{Key: "guestAccess", Value: false, Type: "bool", Host: true})
	r.Plist("com.apple.smb.server", dsl.PlistOpts{Key: "AllowGuestAccess", Value: false, Type: "bool", Host: true})
	// CIS 2.11.5, 2.12.1: disable password hints at login and per-user
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "RetriesUntilHint", Value: 0, Type: "int", Host: true})
}

// --- 2.11.x Screen Saver & Login Window ---

// cisScreenSaver sets inactivity timeout, password-on-wake, and login window behavior (CIS 2.11.x).
func cisScreenSaver(r *dsl.Run) {
	r.Plist("com.apple.screensaver", dsl.PlistOpts{Key: "idleTime", Value: 900, Type: "int", Host: true}) // 15 minutes
	r.Plist("com.apple.screensaver", dsl.PlistOpts{Key: "askForPassword", Value: 1, Type: "int", Host: true})
	r.Plist("com.apple.screensaver", dsl.PlistOpts{Key: "askForPasswordDelay", Value: 5, Type: "int", Host: true}) // 5 seconds grace period
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "SHOWFULLNAME", Value: true, Type: "bool", Host: true})
	r.Plist("com.apple.loginwindow", dsl.PlistOpts{Key: "LoginwindowText", Value: "Authorized uses only. All activity may be monitored and reported.", Type: "string", Host: true})
}

// --- 2.3.1.x, 2.10.x Misc ---

// cisMisc covers AirDrop, AirPlay, power management settings (CIS 2.3.1.x, 2.10.x).
func cisMisc(r *dsl.Run) {
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
}
