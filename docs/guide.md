# Guide

How to write, register, compose, and test Converge blueprints, plus a complete reference for every built-in resource type.

---

## What Is a Blueprint

A blueprint is a Go function with the signature `func(r *dsl.Run)`. Inside the function, you call resource methods on `r` to declare what the system should look like. The function is registered by name with `app.Register()` and compiled into the binary.

Blueprints don't *do* anything directly. They declare intent. The engine diffs current state vs. declared state and applies only what's needed.

---

## Writing a Blueprint

```go
package myblueprint

import "github.com/TsekNet/converge/dsl"

func Blueprint(r *dsl.Run) {
    r.File("/etc/motd", dsl.FileOpts{
        Content: "Welcome to a Converge-managed system\n",
        Mode:    0644,
    })

    r.Package("git", dsl.PackageOpts{
        State: dsl.Present,
    })

    r.Service("sshd", dsl.ServiceOpts{
        State:  dsl.Running,
        Enable: true,
    })
}
```

This declares three things: a file with specific content and permissions, an installed package, and a running enabled service.

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
converge plan myblueprint       # dry-run, show what would change
converge apply myblueprint      # apply the blueprint
```

You can register as many blueprints as you want. Each becomes a subcommand target.

---

## Platform-Conditional Logic

Blueprints are Go code, so platform branching is just `if` statements:

```go
func Blueprint(r *dsl.Run) {
    p := r.Platform()

    if p.OS == "linux" {
        r.File("/etc/motd", dsl.FileOpts{
            Content: "Linux host managed by Converge\n",
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

The compiler enforces this -- you can't call `r.Registry()` from a Linux-tagged file. No runtime "skipped" messages.

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

All option structs share a common field:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Critical` | `bool` | `true` | If `true`, failure aborts the run. Set `false` for best-effort. |

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

### Exec

Run arbitrary commands. Use sparingly -- prefer declarative resources when they exist.

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

**API:** `golang.org/x/sys/windows/registry` -- no `reg.exe`.

**Idempotency:** Reads current value via type-appropriate getter and compares. Creates intermediate keys if needed without disturbing existing sibling values.

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

**API:** `netapi32.dll` `NetUserModalsGet/Set` -- no `secedit.exe`.

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

**API:** `advapi32.dll` `AuditQuerySystemPolicy/AuditSetSystemPolicy` -- no `auditpol.exe`.

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

**API:** Direct file I/O to `/proc/sys/` -- no `sysctl` command.

**Idempotency:** Reads the live kernel value from `/proc/sys/<key>` and compares. Writes only on mismatch.

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

**API:** `howett.net/plist` for binary plist encode/decode -- no `defaults` command.

**Idempotency:** Reads the plist file, decodes, and compares the key's current value. Writes only on mismatch using read-modify-write to preserve other keys.
