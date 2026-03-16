# Examples

How to write, register, compose, and test Converge blueprints, plus a complete reference for every built-in resource type with per-platform examples.

For real-world blueprints, see the [blueprints/](../blueprints/) directory, including [CIS L1 benchmarks](../blueprints/cis/) for Windows, Ubuntu, and macOS.

---

## What Is a Blueprint

A blueprint is a Go function with the signature `func(r *dsl.Run)`. Inside the function, you call resource methods on `r` to declare what the system should look like. The function is registered by name with `app.Register()` and compiled into the binary.

Blueprints don't *do* anything directly. They declare intent. The engine diffs current state vs. declared state and applies only what's needed.

---

## Writing a Blueprint

```go
package blueprints

import "github.com/TsekNet/converge/dsl"

func Baseline(r *dsl.Run) {
    p := r.Platform()

    // Cross-platform: ensure git is installed everywhere
    r.Package("git", dsl.PackageOpts{State: dsl.Present})

    // Platform-specific system banner
    switch p.OS {
    case "linux":
        r.File("/etc/motd", dsl.FileOpts{
            Content: "Managed by Converge\n",
            Mode:    0644,
        })
        r.Service("sshd", dsl.ServiceOpts{State: dsl.Running, Enable: true})
    case "darwin":
        r.File("/etc/motd", dsl.FileOpts{
            Content: "Managed by Converge\n",
            Mode:    0644,
        })
    case "windows":
        r.File(`C:\ProgramData\Converge\motd.txt`, dsl.FileOpts{
            Content: "Managed by Converge\r\n",
        })
    }
}
```

This declares a cross-platform package, platform-specific banner files, and a Linux service. The engine handles the rest.

Here's a more complete example showing cross-platform logic, encrypted secrets, and canary rollouts:

```go
package blueprints

import "github.com/TsekNet/converge/dsl"

func Baseline(r *dsl.Run) {
    p := r.Platform()

    // Cross-platform: install git everywhere
    r.Package("git", dsl.PackageOpts{State: dsl.Present})

    // Cross-platform: allow SSH inbound
    r.Firewall("Allow SSH", dsl.FirewallOpts{Port: 22, Action: "allow"})

    // Cross-platform: use encrypted secrets
    enrollSecret := r.Secret("fleet.enroll_secret")

    // Platform-specific config files
    switch p.OS {
    case "linux":
        r.File("/etc/default/orbit", dsl.FileOpts{
            Content: "ORBIT_ENROLL_SECRET=" + enrollSecret + "\n",
            Mode:    0600,
        })
        r.Service("orbit", dsl.ServiceOpts{State: dsl.Running, Enable: true})

    case "darwin":
        r.File("/etc/orbit/secret", dsl.FileOpts{
            Content: enrollSecret,
            Mode:    0600,
        })

    case "windows":
        r.File(`C:\Program Files\Orbit\secret`, dsl.FileOpts{
            Content: enrollSecret,
        })
    }

    // Canary rollout: 10% of fleet gets the new agent
    if r.InShard(10) {
        r.Package("new-monitoring-agent", dsl.PackageOpts{State: dsl.Present})
    }
}
```

---

## Registering in main.go

```go
package main

import (
    "github.com/TsekNet/converge/dsl"
    "github.com/myorg/myinfra/blueprints/myblueprint"
)

func main() {
    app := dsl.New()
    app.Register("myblueprint", "My system blueprint", myblueprint.Blueprint)
    app.Execute()
}
```

`Register` takes three arguments: name, description, and blueprint function. After building:

```bash
converge plan myblueprint              # dry-run, show what would change
sudo converge serve myblueprint        # run as persistent daemon
sudo converge serve myblueprint --timeout 1s # converge and exit
```

You can register as many blueprints as you want. Each becomes a subcommand target.

---

## Platform-Conditional Logic

Blueprints are Go code, so platform branching is just `if` statements:

```go
func Blueprint(r *dsl.Run) {
    p := r.Platform()

    switch p.OS {
    case "linux":
        r.File("/etc/motd", dsl.FileOpts{
            Content: "Linux host managed by Converge\n",
        })
    case "darwin":
        r.File("/etc/motd", dsl.FileOpts{
            Content: "macOS host managed by Converge\n",
        })
    case "windows":
        r.File(`C:\ProgramData\Converge\motd.txt`, dsl.FileOpts{
            Content: "Windows host managed by Converge\r\n",
        })
    }
}
```

`r.Platform()` returns a `platform.Info` struct with `OS`, `Distro`, `Arch`, `PkgManager`, and `InitSystem`.

For platform-specific resources like `r.Registry()` (Windows), `r.Sysctl()` (Linux), or `r.Plist()` (macOS), use Go build tags on the blueprint file itself:

```go
//go:build windows

package blueprints

import "github.com/TsekNet/converge/dsl"

func Windows(r *dsl.Run) {
    r.Registry(`HKLM\SOFTWARE\MyOrg\Converge`, dsl.RegistryOpts{
        Value: "Managed",
        Type:  "string",
        Data:  "true",
    })
}
```

The compiler enforces this: you can't call `r.Registry()` from a Linux-tagged file. No runtime "skipped" messages.

---

## Explicit Dependencies with DependsOn

Auto-edges handle most dependency relationships automatically (Service->Package, File->parent Dir). For dependencies that auto-edges cannot detect, use `DependsOn`:

```go
func Blueprint(r *dsl.Run) {
    r.Package("postgresql", dsl.PackageOpts{State: dsl.Present})

    r.Exec("db-migrate", dsl.ExecOpts{
        Command:   "/usr/bin/db-migrate",
        OnlyIf:    "test -f /var/lib/myapp/.migration-done",
        Meta: dsl.ResourceMeta{DependsOn: []string{"package:postgresql"}},
    })

    r.Service("myapp", dsl.ServiceOpts{
        State:     dsl.Running,
        Enable:    true,
        Meta: dsl.ResourceMeta{DependsOn: []string{"exec:db-migrate"}},
    })
}
```

This creates a three-layer DAG: `package:postgresql` -> `exec:db-migrate` -> `service:myapp`. In daemon mode, if the postgresql package drifts (e.g. gets uninstalled), converge reinstalls it and then walks the DAG forward to re-check the migration and service.

---

## Daemon Mode (`converge serve`)

The daemon is the primary way to run converge in production. It performs an initial convergence, then watches for drift via OS events:

```bash
# Linux
sudo converge serve baseline

# macOS
sudo converge serve baseline

# Windows (elevated PowerShell)
converge.exe serve baseline
```

What happens:

1. The DAG is built with auto-edges and explicit `DependsOn` relationships
2. Initial convergence runs all resources in topological order
3. Per-resource watchers start (inotify for files, dbus for services, etc.)
4. When drift is detected, the drifted resource is re-checked and its DAG dependents are re-evaluated
5. On failure, exponential backoff retries up to `--max-retries` (default 3)

For CI/CD or image baking (Packer), use `--once` to converge and exit:

```bash
# CI pipeline
sudo converge serve baseline --timeout 1s --detailed-exit-codes

# Packer provisioner
sudo converge serve cis --timeout 1s

# Packer: converge and exit after 60s of stability
sudo converge serve baseline --timeout 60s
```

---

## Blueprint Composition

Split large blueprints and compose with `r.Include()`:

```go
func Blueprint(r *dsl.Run) {
    r.Include("base")        // calls the "base" blueprint
    r.Include("security")    // calls the "security" blueprint
    r.Include("monitoring")  // calls the "monitoring" blueprint
}
```

`Include` calls another registered blueprint by name, injecting its resources into the current Run.

---

## Testing Blueprints

Blueprints are Go functions. Test them with `go test`:

```go
package myblueprint_test

import (
    "testing"
    "github.com/TsekNet/converge/dsl/testing/mock"
    "github.com/myorg/myinfra/blueprints/myblueprint"
)

func TestBlueprint(t *testing.T) {
    r := mock.NewRun()
    myblueprint.Blueprint(r)

    if !r.HasFile("/etc/motd") {
        t.Error("expected /etc/motd to be declared")
    }

    if !r.HasPackage("git") {
        t.Error("expected git package to be declared")
    }
}
```

No containers, no VMs, no network calls.

---

## Resource Reference

All option structs share these common fields via `dsl.ResourceMeta`:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Critical` | `bool` | `true` | If `true`, failure aborts the run. Set `false` for best-effort. |
| `DependsOn` | `[]string` | `nil` | Explicit resource IDs this resource depends on. Complements auto-edges. |
| `Noop` | `bool` | `false` | Per-resource dry-run: Check only, skip Apply. |
| `Retry` | `int` | `0` | Per-resource max retries. 0 = use daemon default (--max-retries). |
| `Limit` | `float64` | `0` | Per-resource rate limit (events/sec). 0 = use daemon default. |
| `AutoEdge` | `*bool` | `nil` | Set to `ptrFalse` to disable auto-edges for this resource. |
| `AutoGroup` | `*bool` | `nil` | Set to `ptrFalse` to disable auto-grouping for this resource. |

**Auto-edges:** Dependencies are detected automatically: `service:X` depends on `package:X`, files depend on parent directories, services depend on config files. Use `DependsOn` for dependencies auto-edges cannot detect.

**Auto-grouping:** Package resources with the same manager and state are automatically batched into a single install/remove transaction (e.g., `apt install git curl neovim` instead of three separate calls). Set `AutoGroup: ptrFalse` on a package to opt out.

Use `DependsOn` for dependencies auto-edges cannot detect:

```go
r.Exec("migrate", dsl.ExecOpts{
    Command:   "/usr/bin/db-migrate",
    Meta: dsl.ResourceMeta{DependsOn: []string{"package:postgresql"}},
})
```

### File

Manage file content, permissions, and ownership.

```go
r.File(path string, opts dsl.FileOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Content` | `string` | `""` | Desired file content. Empty string creates an empty file. |
| `Mode` | `os.FileMode` | `0644` | POSIX permission bits. Ignored on Windows. |
| `Owner` | `string` | `""` | File owner (username). No-op if empty. |
| `Group` | `string` | `""` | File group. No-op if empty. |
| `Append` | `bool` | `false` | If `true`, appends `Content` instead of replacing. |

**Platform behavior:** Full support on Linux/macOS. On Windows, `Mode`/`Owner`/`Group` are ignored.

**Idempotency:** Compares content byte-for-byte and stat metadata. No write if current state matches.

**Examples:**

```go
// Linux: system banner
r.File("/etc/motd", dsl.FileOpts{
    Content: "Managed by Converge\n",
    Mode:    0644,
})

// macOS: LaunchDaemon plist
r.File("/Library/LaunchDaemons/com.example.agent.plist", dsl.FileOpts{
    Content: agentPlist,
    Mode:    0644,
    Owner:   "root",
    Group:   "wheel",
})

// Windows: config file (Mode/Owner/Group ignored)
r.File(`C:\ProgramData\MyApp\config.json`, dsl.FileOpts{
    Content: `{"managed": true}`,
})

// Append to an existing file (all platforms)
r.File("/etc/hosts", dsl.FileOpts{
    Content: "10.0.0.5 internal.example.com\n",
    Append:  true,
})
```

### Package

Install or remove packages via the detected system package manager.

```go
r.Package(name string, opts dsl.PackageOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `State` | `dsl.ResourceState` | `dsl.Present` | `Present` to install, `Absent` to remove. |

| Platform | Package Manager |
|----------|----------------|
| Linux (Debian/Ubuntu) | `apt` |
| Linux (RHEL/Fedora) | `dnf` / `yum` |
| Linux (SUSE) | `zypper` |
| Linux (Alpine) | `apk` |
| Linux (Arch) | `pacman` |
| macOS | `brew` |
| Windows | `choco` / `winget` |

**Idempotency:** Queries the package manager before acting. Installing an already-installed package or removing a missing one is a no-op.

**Examples:**

```go
// Linux (Ubuntu/Debian): install via apt
r.Package("curl", dsl.PackageOpts{State: dsl.Present})

// Linux (Fedora): install via dnf (auto-detected)
r.Package("httpd", dsl.PackageOpts{State: dsl.Present})

// macOS: install via brew
r.Package("jq", dsl.PackageOpts{State: dsl.Present})

// Windows: install via choco
r.Package("7zip", dsl.PackageOpts{State: dsl.Present})

// Remove a package (all platforms)
r.Package("telnet", dsl.PackageOpts{State: dsl.Absent})
```

### Service

Manage service runtime state, boot-time enablement, and startup type.

```go
r.Service(name string, opts dsl.ServiceOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `State` | `dsl.ServiceState` | `dsl.Running` | `Running` or `Stopped`. |
| `Enable` | `bool` | `false` | If `true`, enable the service to start at boot. |
| `StartupType` | `string` | `""` | Windows SCM startup type: `"auto"`, `"delayed-auto"`, `"manual"`, `"disabled"`. |

| Platform | Init System | API |
|----------|-------------|-----|
| Linux | `systemd` | `systemctl` |
| macOS | `launchd` | stub (not yet implemented) |
| Windows | Windows SCM | `golang.org/x/sys/windows/svc/mgr` (native Win32) |

**Idempotency:** Checks current state before acting. Starting a running service is a no-op.

**Examples:**

```go
// Linux: enable and start sshd via systemd
r.Service("sshd", dsl.ServiceOpts{
    State:  dsl.Running,
    Enable: true,
})

// Linux: stop and disable a service
r.Service("cups", dsl.ServiceOpts{
    State:  dsl.Stopped,
    Enable: false,
})

// Windows: set a service to delayed auto-start
r.Service("wuauserv", dsl.ServiceOpts{
    State:       dsl.Running,
    Enable:      true,
    StartupType: "delayed-auto",
})

// Windows: disable Windows Update service
r.Service("wuauserv", dsl.ServiceOpts{
    State:       dsl.Stopped,
    StartupType: "disabled",
})
```

### Exec

Run arbitrary commands. Use sparingly: prefer declarative resources when they exist.

```go
r.Exec(name string, opts dsl.ExecOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Command` | `string` | `""` | The command to execute. Required. |
| `Args` | `[]string` | `nil` | Arguments passed to the command. |
| `OnlyIf` | `string` | `""` | Guard command. If exits 0, `Command` is skipped (state already correct). |
| `Retries` | `int` | `0` | Number of retry attempts on failure. |
| `RetryDelay` | `time.Duration` | `0` | Delay between retries. |

**Idempotency:** Not inherently idempotent. Always provide an `OnlyIf` command to make it conditional.

**Examples:**

```go
// Linux: run a script only if a marker file is missing
r.Exec("bootstrap", dsl.ExecOpts{
    Command: "/usr/local/bin/bootstrap.sh",
    OnlyIf:  "test -f /var/lib/myapp/.bootstrapped",
})

// macOS: flush DNS cache
r.Exec("flush-dns", dsl.ExecOpts{
    Command: "/usr/bin/dscacheutil",
    Args:    []string{"-flushcache"},
})

// Windows: enable TLS 1.2 (OnlyIf checks if already set)
r.Exec("enable-tls12", dsl.ExecOpts{
    Command: "powershell.exe",
    Args:    []string{"-NoProfile", "-Command", "New-ItemProperty -Path 'HKLM:\\SYSTEM\\...' -Name Enabled -Value 1"},
    OnlyIf:  "powershell.exe -NoProfile -Command \"(Get-ItemProperty 'HKLM:\\SYSTEM\\...').Enabled -eq 1\"",
})

// Retry on transient failure (all platforms)
r.Exec("download-agent", dsl.ExecOpts{
    Command:    "curl",
    Args:       []string{"-fsSL", "-o", "/tmp/agent.tar.gz", "https://example.com/agent.tar.gz"},
    OnlyIf:     "test -f /tmp/agent.tar.gz",
    Retries:    3,
    RetryDelay: 5 * time.Second,
})
```

### User

Create and manage local user accounts and group membership.

```go
r.User(name string, opts dsl.UserOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Groups` | `[]string` | `nil` | Supplementary groups the user should belong to. |
| `Shell` | `string` | `""` | Login shell path. No-op if empty. |
| `Home` | `string` | `""` | Home directory path. Uses OS default if empty. |
| `System` | `bool` | `false` | If `true`, creates a system account (low UID, no home by default). |

| Platform | Tooling |
|----------|---------|
| Linux | `useradd` / `usermod` |
| macOS | `dscl` / Directory Services |
| Windows | `net user` / `net localgroup` (`Shell` and `System` ignored) |

**Idempotency:** Creates user if missing. Modifies only divergent attributes if user exists.

**Examples:**

```go
// Linux: create app user with specific shell and groups
r.User("deploy", dsl.UserOpts{
    Shell:  "/bin/bash",
    Home:   "/home/deploy",
    Groups: []string{"docker", "sudo"},
})

// Linux: system account (low UID, no home)
r.User("prometheus", dsl.UserOpts{
    System: true,
    Shell:  "/usr/sbin/nologin",
})

// macOS: add user to admin group
r.User("admin", dsl.UserOpts{
    Groups: []string{"admin", "staff"},
})

// Windows: add user to local Administrators (Shell/System ignored)
r.User("svc-agent", dsl.UserOpts{
    Groups: []string{"Administrators"},
})
```

### Registry (Windows only)

Manage Windows registry keys and values via native Win32 API. Available only in `//go:build windows` blueprints.

```go
r.Registry(key string, opts dsl.RegistryOpts)
```

The `key` is the full registry path, e.g., `HKLM\SOFTWARE\MyApp`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Value` | `string` | `""` | The registry value name under the key. |
| `Type` | `string` | `"sz"` | `"sz"`, `"dword"`, `"qword"`, `"expandstring"`, `"multistring"`, `"binary"`. Also accepts `"REG_DWORD"` etc. |
| `Data` | `any` | `nil` | The data to set. Type must match `Type`. |
| `State` | `dsl.ResourceState` | `dsl.Present` | `Present` to create/set, `Absent` to delete the value. |

**Supported root keys:** `HKLM`, `HKCU`, `HKCR`, `HKU`, `HKCC` (and their long forms like `HKEY_LOCAL_MACHINE`).

**API:** `golang.org/x/sys/windows/registry`: no `reg.exe`.

**Idempotency:** Reads current value via type-appropriate getter and compares. Creates intermediate keys if needed without disturbing existing sibling values.

**Examples:**

```go
//go:build windows

// Set a string value
r.Registry(`HKLM\SOFTWARE\MyOrg`, dsl.RegistryOpts{
    Value: "Environment",
    Type:  "sz",
    Data:  "production",
})

// Set a DWORD (integer) value
r.Registry(`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`, dsl.RegistryOpts{
    Value: "NoAutoUpdate",
    Type:  "dword",
    Data:  uint32(1),
})

// Delete a registry value
r.Registry(`HKLM\SOFTWARE\MyOrg`, dsl.RegistryOpts{
    Value: "DeprecatedSetting",
    State: dsl.Absent,
})
```

### SecurityPolicy (Windows only)

Manage Windows local security policy settings via native Win32 APIs. Available only in `//go:build windows` blueprints.

```go
r.SecurityPolicy(name string, opts dsl.SecurityPolicyOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Category` | `string` | `""` | `"password"` or `"lockout"`. Required. |
| `Key` | `string` | `""` | Setting name. Required. |
| `Value` | `string` | `""` | Desired value as string. |

**Password policy keys** (`Category: "password"`):

| Key | Description |
|-----|-------------|
| `MinimumPasswordLength` | Minimum password length |
| `MaximumPasswordAge` | Maximum password age (seconds) |
| `MinimumPasswordAge` | Minimum password age (seconds) |
| `PasswordHistorySize` | Number of remembered passwords |
| `ForceLogoff` | Force logoff time (seconds) |

**Lockout policy keys** (`Category: "lockout"`):

| Key | Description |
|-----|-------------|
| `LockoutThreshold` | Failed logon attempts before lockout |
| `LockoutDuration` | Lockout duration (seconds) |
| `LockoutObservationWindow` | Observation window (seconds) |

**API:** `netapi32.dll` `NetUserModalsGet/Set`: no `secedit.exe`.

**Examples:**

```go
//go:build windows

// Require 12-character minimum passwords
r.SecurityPolicy("min-password-length", dsl.SecurityPolicyOpts{
    Category: "password",
    Key:      "MinimumPasswordLength",
    Value:    "12",
})

// Lock accounts after 5 failed attempts for 30 minutes
r.SecurityPolicy("lockout-threshold", dsl.SecurityPolicyOpts{
    Category: "lockout",
    Key:      "LockoutThreshold",
    Value:    "5",
})
r.SecurityPolicy("lockout-duration", dsl.SecurityPolicyOpts{
    Category: "lockout",
    Key:      "LockoutDuration",
    Value:    "1800",
})
```

### AuditPolicy (Windows only)

Manage Windows advanced audit policy via native Win32 APIs. Available only in `//go:build windows` blueprints.

```go
r.AuditPolicy(name string, opts dsl.AuditPolicyOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Subcategory` | `string` | `""` | Audit subcategory name (case-insensitive). Required. |
| `Success` | `bool` | `false` | Enable success auditing. |
| `Failure` | `bool` | `false` | Enable failure auditing. |

**Supported subcategories** (59 total, organized by category):

| Category | Subcategories |
|----------|--------------|
| **Account Logon** | Credential Validation, Kerberos Authentication Service, Kerberos Service Ticket Operations, Other Account Logon Events |
| **Account Management** | Application Group Management, Computer Account Management, Distribution Group Management, Other Account Management Events, Security Group Management, User Account Management |
| **Detailed Tracking** | DPAPI Activity, Plug and Play Events, Process Creation, Process Termination, RPC Events, Token Right Adjusted Events |
| **DS Access** | Directory Service Access, Directory Service Changes, Directory Service Replication, Detailed Directory Service Replication |
| **Logon/Logoff** | Account Lockout, Group Membership, IPsec Extended Mode, IPsec Main Mode, IPsec Quick Mode, Logoff, Logon, Network Policy Server, Other Logon/Logoff Events, Special Logon, User / Device Claims |
| **Object Access** | Application Generated, Central Policy Staging, Certification Services, Detailed File Share, File Share, File System, Filtering Platform Connection, Filtering Platform Packet Drop, Handle Manipulation, Kernel Object, Other Object Access Events, Registry, Removable Storage, SAM |
| **Policy Change** | Audit Policy Change, Authentication Policy Change, Authorization Policy Change, Filtering Platform Policy Change, MPSSVC Rule-Level Policy Change, Other Policy Change Events |
| **Privilege Use** | Non Sensitive Privilege Use, Other Privilege Use Events, Sensitive Privilege Use |
| **System** | IPsec Driver, Other System Events, Security State Change, Security System Extension, System Integrity |

**API:** `advapi32.dll` `AuditQuerySystemPolicy/AuditSetSystemPolicy`: no `auditpol.exe`.

**Examples:**

```go
//go:build windows

// Audit successful and failed logon attempts
r.AuditPolicy("audit-logon", dsl.AuditPolicyOpts{
    Subcategory: "Logon",
    Success:     true,
    Failure:     true,
})

// Audit process creation (for security monitoring)
r.AuditPolicy("audit-process-creation", dsl.AuditPolicyOpts{
    Subcategory: "Process Creation",
    Success:     true,
})
```

### Sysctl (Linux only)

Manage Linux kernel parameters via `/proc/sys/`. Available only in `//go:build linux` blueprints.

```go
r.Sysctl(key string, opts dsl.SysctlOpts)
```

The `key` uses dotted notation, e.g., `net.ipv4.ip_forward`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Value` | `string` | `""` | Desired kernel parameter value. Required. |
| `Persist` | `bool` | `true` | If `true`, writes to `/etc/sysctl.d/99-converge.conf` so the setting survives reboots. |

**API:** Direct file I/O to `/proc/sys/`: no `sysctl` command.

**Idempotency:** Reads the live kernel value from `/proc/sys/<key>` and compares. Writes only on mismatch.

**Examples:**

```go
//go:build linux

// Enable IP forwarding
r.Sysctl("net.ipv4.ip_forward", dsl.SysctlOpts{
    Value: "1",
})

// Harden TCP SYN cookies
r.Sysctl("net.ipv4.tcp_syncookies", dsl.SysctlOpts{
    Value: "1",
})

// Set without persisting to disk (takes effect until reboot only)
r.Sysctl("vm.swappiness", dsl.SysctlOpts{
    Value:   "10",
    Persist: false,
})
```

### Plist (macOS only)

Manage macOS preference domain keys via native binary plist encoding. Available only in `//go:build darwin` blueprints.

```go
r.Plist(domain string, opts dsl.PlistOpts)
```

The `domain` is the preference domain, e.g., `com.apple.screensaver`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Key` | `string` | `""` | The preference key to set. Required. |
| `Value` | `any` | `nil` | Desired value. Type inferred from Go type or `Type` field. |
| `Type` | `string` | `""` | Explicit type hint: `"bool"`, `"int"`, `"float"`, `"string"`. |
| `Host` | `bool` | `false` | If `true`, targets `/Library/Preferences` (system-wide). If `false`, targets `~/Library/Preferences`. |

**API:** `howett.net/plist` for binary plist encode/decode: no `defaults` command.

**Idempotency:** Reads the plist file, decodes, and compares the key's current value. Writes only on mismatch using read-modify-write to preserve other keys.

**Examples:**

```go
//go:build darwin

// Disable screensaver password grace period (system-wide)
r.Plist("com.apple.screensaver", dsl.PlistOpts{
    Key:   "askForPasswordDelay",
    Value: 0,
    Type:  "int",
    Host:  true,
})

// Set Finder preferences (user-level)
r.Plist("com.apple.finder", dsl.PlistOpts{
    Key:   "AppleShowAllExtensions",
    Value: true,
    Type:  "bool",
})

// Disable Gatekeeper assessments
r.Plist("com.apple.LaunchServices", dsl.PlistOpts{
    Key:   "LSQuarantine",
    Value: false,
    Type:  "bool",
    Host:  true,
})
```

### Firewall

Manage host firewall rules. Cross-platform: available in all blueprints.

```go
r.Firewall(name string, opts dsl.FirewallOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Port` | `int` | (required) | Port number (1-65535). |
| `Protocol` | `string` | `"tcp"` | `"tcp"` or `"udp"`. |
| `Direction` | `string` | `"inbound"` | `"inbound"` or `"outbound"`. |
| `Action` | `string` | `"allow"` | `"allow"` or `"block"`. |
| `Source` | `string` | `""` | Source IP or CIDR. Empty = any. |
| `Dest` | `string` | `""` | Destination IP or CIDR. Empty = any. |
| `State` | `dsl.ResourceState` | `dsl.Present` | `Present` to create, `Absent` to delete. |

| Platform | Backend | API |
|----------|---------|-----|
| Linux | nftables (IPv4 only) | `github.com/google/nftables` (netlink) |
| macOS | pf (anchor-based) | `/etc/pf.anchors/converge` + `pfctl` reload |
| Windows | Windows Firewall (registry) | `HKLM\...\FirewallRules` + SCM notify |

**Input validation:** All fields are validated at construction time. Invalid names, out-of-range ports, and malformed source/dest addresses are rejected with a panic.

**Idempotency:** Checks if a matching rule exists before creating. On Linux, rules are identified by nftables UserData tag. On macOS, by a pf comment tag. On Windows, by the registry value name and content.

**Examples:**

```go
// Allow SSH inbound (all platforms)
r.Firewall("Allow SSH", dsl.FirewallOpts{
    Port:     22,
    Protocol: "tcp",
    Action:   "allow",
})

// Block outbound to a specific port
r.Firewall("Block SMTP out", dsl.FirewallOpts{
    Port:      25,
    Direction: "outbound",
    Action:    "block",
})

// Allow inbound from a specific subnet only
r.Firewall("Allow monitoring", dsl.FirewallOpts{
    Port:   9090,
    Source: "10.0.0.0/8",
    Action: "allow",
})

// Remove a rule that was previously managed
r.Firewall("Legacy RDP", dsl.FirewallOpts{
    Port:  3389,
    State: dsl.Absent,
})
```

### InShard

Percentage-based rollout sharding. Not a resource (no Check/Apply), but a DSL helper for conditional logic in blueprints.

```go
// Roll out a new package to 10% of the fleet
if r.InShard(10) {
    r.Package("new-agent", dsl.PackageOpts{State: dsl.Present})
}

// Canary a config change: 5% first, then expand to 50%, then 100%
if r.InShard(5) {
    r.File("/etc/myapp/experimental.conf", dsl.FileOpts{
        Content: "feature_v2=true\n",
    })
}

// Use InShardWithSerial in tests to verify shard logic
// (the serial is normalized the same way as InShard)
if r.InShardWithSerial(10, "TEST-SERIAL") {
    // This runs deterministically in tests
}
```

The shard is computed from the first 7 characters of the hardware serial number via SHA-256. The same machine always lands in the same bucket across runs. Known placeholder serials (`Not Specified`, `To Be Filled By O.E.M.`, etc.) are rejected, causing `InShard` to return `false`.

| Method | Signature | Description |
|--------|-----------|-------------|
| `InShard` | `(percent int) bool` | Uses auto-detected hardware serial. |
| `InShardWithSerial` | `(percent int, serial string) bool` | Uses an explicit serial (for testing). The serial is normalized identically to `InShard` (trimmed, placeholder-checked, truncated to 7 chars). |
| `ShardBucket` | `(serial string) uint64` | Returns the shard bucket [0, 100) for a given serial. Package-level function, useful for testing. |

| Platform | Serial Source |
|----------|--------------|
| Linux | `/sys/class/dmi/id/product_serial` |
| macOS | `kern.uuid` via `/usr/sbin/sysctl` (hardware UUID) |
| Windows | `HKLM\HARDWARE\DESCRIPTION\System\BIOS\SerialNumber` registry |

### Secret / Encrypted Config

Retrieve config values from Go maps registered at compile time. Values wrapped in `ENC[AES256:...]` are decrypted transparently with AES-256-GCM.

```go
// In a config file (e.g., config.go):
func init() {
    dsl.RegisterConfig(map[string]any{
        "fleet": map[string]any{
            "server_url":    "https://fleet.example.com",
            "enroll_secret": "ENC[AES256:base64ciphertext...]",
        },
    })
}

// In a blueprint:
func Blueprint(r *dsl.Run) {
    secret := r.Secret("fleet.enroll_secret") // decrypted automatically
    url := r.Secret("fleet.server_url")       // plain value returned as-is
}
```

**Key management:** Call `dsl.SetConfigKey("your-key")` once at startup, typically sourced from an environment variable. The key is SHA-256 hashed to 32 bytes, so use a high-entropy random key (not a passphrase). Generate encrypted values with `val, err := dsl.Encrypt("plaintext")`.

**Fail-closed:** If the key is missing or decryption fails, `Secret` returns an empty string (never leaks ciphertext).

**No external deps:** Uses Go stdlib `crypto/aes` + `crypto/cipher` (AES-256-GCM with random nonces).
