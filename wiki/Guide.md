# Guide

**[← Wiki Home](Home)** · [Design](Design) · [CLI](CLI) · [Extending](Extending)

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

    if p.OS == "windows" {
        r.Registry(`HKLM\SOFTWARE\MyOrg\Converge`, dsl.RegistryOpts{
            Value: "Managed",
            Type:  "string",
            Data:  "true",
        })
    }
}
```

`r.Platform()` returns a `platform.Info` struct with `OS`, `Distro`, `Arch`, `PkgManager`, and `InitSystem`.

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
| Windows | `choco` |
| Windows | `winget` |

**Idempotency:** Queries the package manager before acting. Installing an already-installed package or removing a missing one is a no-op.

### Service

Manage service runtime state and boot-time enablement.

```go
r.Service(name string, opts dsl.ServiceOpts)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `State` | `dsl.ServiceState` | `dsl.Running` | `Running` or `Stopped`. |
| `Enable` | `bool` | `false` | If `true`, enable the service to start at boot. |

| Platform | Init System |
|----------|-------------|
| Linux | `systemd` (systemctl) |
| macOS | `launchd` (launchctl) |
| Windows | Windows Service Control Manager (sc.exe) |

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
| `Check` | `string` | `""` | Guard command. If exits 0, `Command` is skipped (state already correct). |
| `Retries` | `int` | `0` | Number of retry attempts on failure. |
| `RetryDelay` | `time.Duration` | `0` | Delay between retries. |

| Platform | Shell |
|----------|-------|
| Linux/macOS | `/bin/sh -c` |
| Windows | `cmd.exe /C` |

**Idempotency:** Not inherently idempotent. Always provide a `Check` command to make it conditional.

**Example:**

```go
r.Exec("reload-nginx", dsl.ExecOpts{
    Command:    "nginx",
    Args:       []string{"-s", "reload"},
    Check:      "nginx -t",
    Retries:    2,
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
| Windows | `net user` / Win32 API (`Shell` and `System` ignored) |

**Idempotency:** Creates user if missing. Modifies only divergent attributes if user exists.

### Registry

Manage Windows registry keys and values. On non-Windows platforms, this resource is a silent no-op.

```go
r.Registry(key string, opts dsl.RegistryOpts)
```

The `key` is the full registry path including value name, e.g., `HKLM\SOFTWARE\MyApp\Setting`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Value` | `string` | `""` | The registry value name under the key. |
| `Type` | `string` | `"string"` | `"string"`, `"dword"`, `"qword"`, `"expandstring"`, `"multistring"`, `"binary"`. |
| `Data` | `interface{}` | `nil` | The data to set. Type must match `Type`. |

**Platform behavior:** Full support on Windows. No-op on Linux/macOS.

**Idempotency:** Reads current value and compares type and data. Creates intermediate keys if needed without disturbing existing sibling values.
