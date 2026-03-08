//go:build windows

package cis

import "github.com/TsekNet/converge/dsl"

// WindowsCIS declares the CIS Microsoft Windows 11 Enterprise L1 benchmark.
// Covers ~242 settings across registry, security policy, audit policy, and services.
func WindowsCIS(r *dsl.Run) {
	cisRegistrySettings(r)
	cisSecurityPolicy(r)
	cisAuditPolicy(r)
	cisServices(r)
}

// cisRegistrySettings enforces ~130 registry DWORD values across CIS sections 18.x.
func cisRegistrySettings(r *dsl.Run) {
	type regSetting struct {
		key, value, regType string
		data                any
	}

	dwordSettings := []regSetting{
		// Account policies
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "NoConnectedUser", "dword", 3},
		// Lock screen (18.1.x)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Personalization`, "NoLockScreenCamera", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Personalization`, "NoLockScreenSlideshow", "dword", 1},
		// LSA hardening (18.4.x)
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "CrashOnAuditFail", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "NoLMHash", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "RestrictAnonymousSAM", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "RestrictAnonymous", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "EveryoneIncludesAnonymous", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "ForceGuest", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "UseMachineId", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "LimitBlankPasswordUse", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "SCENoApplyLegacyAuditPolicy", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0`, "NTLMMinClientSec", "dword", 537395200},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0`, "NTLMMinServerSec", "dword", 537395200},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0`, "allownullsessionfallback", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "LmCompatibilityLevel", "dword", 5},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa\pku2u`, "AllowOnlineID", "dword", 0},
		// SMB signing + network security
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters`, "RequireSecuritySignature", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters`, "EnableSecuritySignature", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters`, "EnablePlainTextPassword", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters`, "RequireSecuritySignature", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters`, "EnableSecuritySignature", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters`, "AutoDisconnect", "dword", 15},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters`, "RestrictNullSessAccess", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters`, "EnableForcedLogOff", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`, "RequireSignOrSeal", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`, "SealSecureChannel", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`, "SignSecureChannel", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`, "DisablePasswordChange", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`, "MaximumPasswordAge", "dword", 30},
		{`HKLM\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters`, "RequireStrongKey", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LDAP`, "LDAPClientIntegrity", "dword", 1},
		// Interactive logon
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "DontDisplayLastUserName", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "DisableCAD", "dword", 0},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "InactivityTimeoutSecs", "dword", 900},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "ScForceOption", "dword", 0},
		{`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon`, "CachedLogonsCount", "dword", 4},
		{`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon`, "PasswordExpiryWarning", "dword", 14},
		{`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon`, "ForceUnlockLogon", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon`, "ScRemoveOption", "dword", 1},
		// UAC (18.9.x)
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "EnableLUA", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "ConsentPromptBehaviorAdmin", "dword", 2},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "ConsentPromptBehaviorUser", "dword", 0},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "EnableInstallerDetection", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "EnableSecureUIAPaths", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "EnableVirtualization", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "PromptOnSecureDesktop", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "FilterAdministratorToken", "dword", 1},
		// AutoPlay / AutoRun (18.9.8)
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer`, "NoDriveTypeAutoRun", "dword", 255},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer`, "NoAutorun", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Explorer`, "NoAutoplayfornonVolume", "dword", 1},
		// BitLocker (18.9.11)
		{`HKLM\SOFTWARE\Policies\Microsoft\FVE`, "UseAdvancedStartup", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\FVE`, "EnableBDEWithNoTPM", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\FVE`, "UseTPM", "dword", 2},
		{`HKLM\SOFTWARE\Policies\Microsoft\FVE`, "UseTPMPIN", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\FVE`, "UseTPMKey", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\FVE`, "UseTPMKeyPIN", "dword", 0},
		// Camera, cloud, credentials
		{`HKLM\SOFTWARE\Policies\Microsoft\Camera`, "AllowCamera", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\CloudContent`, "DisableConsumerAccountStateContent", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\CloudContent`, "DisableWindowsConsumerFeatures", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\CredentialsDelegation`, "AllowProtectedCreds", "dword", 1},
		// Device Guard (18.9.17)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeviceGuard`, "EnableVirtualizationBasedSecurity", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeviceGuard`, "RequirePlatformSecurityFeatures", "dword", 3},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeviceGuard`, "HypervisorEnforcedCodeIntegrity", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeviceGuard`, "LsaCfgFlags", "dword", 1},
		// Telemetry (18.9.17)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "AllowTelemetry", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "DisableOneSettingsDownloads", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "DoNotShowFeedbackNotifications", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "EnableOneSettingsAuditing", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "LimitDiagnosticLogCollection", "dword", 1},
		// Early Launch Antimalware
		{`HKLM\SYSTEM\CurrentControlSet\Policies\EarlyLaunch`, "DriverLoadPolicy", "dword", 3},
		// Event log sizes (18.9.27)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\EventLog\Application`, "MaxSize", "dword", 32768},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\EventLog\Security`, "MaxSize", "dword", 196608},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\EventLog\Setup`, "MaxSize", "dword", 32768},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\EventLog\System`, "MaxSize", "dword", 32768},
		// File Explorer (18.9.31)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Explorer`, "NoDataExecutionPrevention", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Explorer`, "NoHeapTerminationOnCorruption", "dword", 0},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer`, "PreXPSP2ShellProtocolBehavior", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\HomeGroup`, "DisableHomeGroup", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer`, "NoWebServices", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Internet Connection Wizard`, "ExitOnMSICW", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\EventViewer`, "MicrosoftEventVwrDisableLinks", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System\Kerberos\Parameters`, "SupportedEncryptionTypes", "dword", 2147483640},
		// Defender Antivirus (18.9.47)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender`, "DisableAntiSpyware", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender`, "PUAProtection", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection`, "DisableBehaviorMonitoring", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection`, "DisableRealtimeMonitoring", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection`, "DisableIOAVProtection", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection`, "DisableScriptScanning", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Scan`, "DisablePackedExeScanning", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Scan`, "DisableRemovableDriveScanning", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Scan`, "DisableEmailScanning", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Spynet`, "SpynetReporting", "dword", 2},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\MpEngine`, "MpEnablePus", "dword", 1},
		// SmartScreen (18.9.85)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\System`, "EnableSmartScreen", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\System`, "ShellSmartScreenLevel", "dword", 1},
		// Windows Installer
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Installer`, "AlwaysInstallElevated", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Installer`, "EnableUserControl", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\System`, "BlockDomainPicturePassword", "dword", 1},
		// WinRM (18.9.102)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Client`, "AllowBasic", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Client`, "AllowUnencryptedTraffic", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Client`, "AllowDigest", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service`, "AllowBasic", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service`, "AllowUnencryptedTraffic", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service`, "DisableRunAs", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service\WinRS`, "AllowRemoteShellAccess", "dword", 0},
		// Windows Update (18.9.108)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate`, "ManagePreviewBuildsPolicyValue", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`, "NoAutoUpdate", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`, "AUOptions", "dword", 4},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`, "NoAutoRebootWithLoggedOnUsers", "dword", 0},
		// Remote Desktop (18.9.65)
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "fDenyTSConnections", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "fDisableCdm", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "fPromptForPassword", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "fEncryptRPCTraffic", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "MinEncryptionLevel", "dword", 3},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "DeleteTempDirsOnExit", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "PerSessionTempDir", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Internet Explorer\Feeds`, "DisableEnclosureDownload", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "AllowIndexingEncryptedStoresOrItems", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "AllowCortanaAboveLock", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\CurrentVersion\Software Protection Platform`, "NoGenTicket", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Error Reporting`, "Disabled", "dword", 1},
		// PowerShell logging
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\PowerShell\ScriptBlockLogging`, "EnableScriptBlockLogging", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\PowerShell\Transcription`, "EnableTranscripting", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "MSAOptional", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient`, "EnableMulticast", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\System`, "EnableFontProviders", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\LLTD`, "AllowLLTDIOOnDomain", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\LLTD`, "AllowLLTDIOOnPublicNet", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\LLTD`, "EnableRspndr", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\LLTD`, "ProhibitLLTDIOOnPrivateNet", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\LLTD`, "ProhibitRspndrOnPrivateNet", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Peernet`, "Disabled", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\WcmSvc\GroupPolicy`, "fMinimizeConnections", "dword", 3},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Printers`, "DisableWebPnPDownload", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Printers`, "DisableHTTPPrinting", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Policies`, "NtfsDisable8dot3NameCreation", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\kernel`, "DisableExceptionChainValidation", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "fAllowUnsolicited", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, "fAllowToGetHelp", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows NT\Rpc`, "RestrictRemoteClients", "dword", 1},
		{`HKLM\SOFTWARE\Microsoft\WcmSvc\wifinetworkmanager\config`, "AutoConnectAllowedOEM", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\WindowsInkWorkspace`, "AllowWindowsInkWorkspace", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeviceGuard`, "HVCIMATRequired", "dword", 1},
		// Credential protection
		{`HKLM\SYSTEM\CurrentControlSet\Control\SecurityProviders\WDigest`, "UseLogonCredential", "dword", 0},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Lsa`, "DisableDomainCreds", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters`, "NullSessionShares", "dword", 0},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "ShutdownWithoutLogon", "dword", 0},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "ClearPageFileAtShutdown", "dword", 1},
		{`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager`, "ProtectionMode", "dword", 1},
		{`HKLM\SOFTWARE\Policies\Microsoft\Cryptography`, "ForceKeyProtection", "dword", 2},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "UndockWithoutLogon", "dword", 0},
		{`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeviceInstall\Settings`, "AllowRemoteRPC", "dword", 0},
	}

	for _, s := range dwordSettings {
		r.Registry(s.key, dsl.RegistryOpts{
			Value: s.value, Type: s.regType, Data: s.data,
		})
	}
}

// cisSecurityPolicy enforces password and account lockout policy via Win32 NetUserModals API (CIS 1.x).
func cisSecurityPolicy(r *dsl.Run) {
	type secpolSetting struct {
		category, key, value string
	}

	settings := []secpolSetting{
		{"password", "MinimumPasswordLength", "14"},
		{"password", "MaximumPasswordAge", "3628800"}, // 42 days in seconds
		{"password", "MinimumPasswordAge", "86400"},   // 1 day in seconds
		{"password", "PasswordHistorySize", "24"},
		{"lockout", "LockoutThreshold", "5"},
		{"lockout", "LockoutDuration", "900"},          // 15 minutes
		{"lockout", "LockoutObservationWindow", "900"}, // 15 minutes
	}

	for _, s := range settings {
		r.SecurityPolicy(s.key, dsl.SecurityPolicyOpts{
			Category: s.category, Key: s.key, Value: s.value,
		})
	}
}

// cisAuditPolicy enables Success+Failure auditing for all CIS-required subcategories (CIS 17.x).
func cisAuditPolicy(r *dsl.Run) {
	type auditSetting struct {
		subcategory      string
		success, failure bool
	}

	settings := []auditSetting{
		{"Credential Validation", true, true},
		{"Kerberos Authentication Service", true, true},
		{"Kerberos Service Ticket Operations", true, true},
		{"Other Account Logon Events", true, true},
		{"Application Group Management", true, true},
		{"Computer Account Management", true, true},
		{"Distribution Group Management", true, true},
		{"Other Account Management Events", true, true},
		{"Security Group Management", true, true},
		{"User Account Management", true, true},
		{"DPAPI Activity", true, true},
		{"Plug and Play Events", true, false},
		{"Process Creation", true, false},
		{"Process Termination", true, false},
		{"RPC Events", true, true},
		{"Token Right Adjusted Events", true, false},
		{"Directory Service Access", true, true},
		{"Directory Service Changes", true, true},
		{"Directory Service Replication", true, true},
		{"Detailed Directory Service Replication", true, true},
		{"Account Lockout", true, false},
		{"Group Membership", true, false},
		{"Logoff", true, false},
		{"Logon", true, true},
		{"Network Policy Server", true, true},
		{"Other Logon/Logoff Events", true, true},
		{"Special Logon", true, false},
		{"User / Device Claims", true, false},
		{"IPsec Extended Mode", true, true},
		{"IPsec Main Mode", true, true},
		{"IPsec Quick Mode", true, true},
		{"Application Generated", true, true},
		{"Certification Services", true, true},
		{"Detailed File Share", true, true},
		{"File Share", true, true},
		{"File System", true, true},
		{"Filtering Platform Connection", true, true},
		{"Filtering Platform Packet Drop", true, true},
		{"Handle Manipulation", true, true},
		{"Kernel Object", true, true},
		{"Other Object Access Events", true, true},
		{"Registry", true, true},
		{"Removable Storage", true, true},
		{"SAM", true, true},
		{"Central Policy Staging", true, true},
		{"Audit Policy Change", true, true},
		{"Authentication Policy Change", true, true},
		{"Authorization Policy Change", true, true},
		{"Filtering Platform Policy Change", true, true},
		{"MPSSVC Rule-Level Policy Change", true, true},
		{"Other Policy Change Events", true, true},
		{"Non Sensitive Privilege Use", true, true},
		{"Other Privilege Use Events", true, true},
		{"Sensitive Privilege Use", true, true},
		{"IPsec Driver", true, true},
		{"Other System Events", true, true},
		{"Security State Change", true, true},
		{"Security System Extension", true, true},
		{"System Integrity", true, true},
	}

	for _, s := range settings {
		r.AuditPolicy(s.subcategory, dsl.AuditPolicyOpts{
			Subcategory: s.subcategory, Success: s.success, Failure: s.failure,
		})
	}
}

// cisServices disables Windows services not needed on a hardened workstation (CIS 5.x).
func cisServices(r *dsl.Run) {
	disabledServices := []string{
		"MapsBroker", "lfsvc", "IISADMIN", "irmon", "SharedAccess", "lltdsvc",
		"LxssManager", "FTPSVC", "MSiSCSI", "sshd", "PNRPsvc", "p2psvc",
		"p2pimsvc", "PNRPAutoReg", "Spooler", "wercplsupport", "RasAuto",
		"SessionEnv", "TermService", "UmRdpService", "RemoteRegistry",
		"RemoteAccess", "RpcLocator", "SNMP", "SSDPSRV", "upnphost", "WMSvc",
		"WerSvc", "Wecsvc", "WMPNetworkSvc", "icssvc", "WpnService",
		"PushToInstall", "WinRM", "W3SVC", "XboxGipSvc", "XblAuthManager",
		"XblGameSave", "XboxNetApiSvc",
	}

	for _, name := range disabledServices {
		r.Service(name, dsl.ServiceOpts{
			State:       dsl.Stopped,
			StartupType: "disabled",
		})
	}
}
