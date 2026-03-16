# Adding a New Extension

This guide walks through adding a new extension to Converge. Extensions are everything that touches the OS: package managers, init systems, file operations, etc.

---

## Extension Interface

Every extension implements:

```go
type Extension interface {
    ID() string
    Check(ctx context.Context) (*State, error)
    Apply(ctx context.Context) (*Result, error)
    String() string
}
```

- `Check()` reads current state and compares to desired. No root needed.
- `Apply()` makes changes. Requires root.
- `ID()` returns a unique identifier like `file:/etc/motd` or `package:git`.

Optionally implement `CriticalResource` to control whether failure halts the run:

```go
type CriticalResource interface {
    IsCritical() bool
}
```

### Daemon Mode: Watcher and Poller

In daemon mode (`converge serve`), extensions can implement optional interfaces for drift detection:

```go
// Watcher blocks on native OS events (inotify, kqueue, dbus, etc.)
// and sends events when the resource may have drifted.
type Watcher interface {
    Watch(ctx context.Context, events chan<- Event) error
}

// Poller overrides the default poll interval for resources without
// native OS event support.
type Poller interface {
    PollInterval() time.Duration
}
```

**When to implement Watcher:** if your resource type has a native OS mechanism for change notification (file system events, D-Bus signals, registry change notifications). See `extensions/file/watch_linux.go` for a reference implementation using inotify.

**When to implement Poller:** if your resource type has no native events but needs a custom poll frequency. For example, packages poll every 5 minutes since package state rarely changes externally.

Extensions implementing neither fall back to the daemon's default poll interval (30 seconds).

### Event Struct Fields

| Field | Type | Description |
|-------|------|-------------|
| `ResourceID` | `string` | Unique identifier of the resource that triggered the event (e.g. `file:/etc/motd`) |
| `Kind` | `EventKind` | How the event was generated (see constants below) |
| `Detail` | `string` | Human-readable context, such as `"inotify"`, `"kqueue"`, or `"RegNotifyChangeKeyValue"` |
| `Time` | `time.Time` | Timestamp when the event was created |

### EventKind Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `EventWatch` | `0` | OS-level watcher detected a change |
| `EventPoll` | `1` | Periodic poll detected drift |
| `EventRetry` | `2` | Scheduled retry after a previous failure |

---

## Example: Adding a New Package Manager (dnf)

### 1. Create the file

Create `extensions/pkg/dnf.go`:

```go
package pkg

import (
    "context"
    "fmt"
    "os/exec"
)

type dnfManager struct{}

func (d *dnfManager) Name() string { return "dnf" }

func (d *dnfManager) IsInstalled(ctx context.Context, name string) (bool, error) {
    cmd := exec.CommandContext(ctx, "rpm", "-q", name)
    err := cmd.Run()
    return err == nil, nil
}

func (d *dnfManager) Install(ctx context.Context, name string) error {
    cmd := exec.CommandContext(ctx, "dnf", "install", "-y", name)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("dnf install %s: %s: %w", name, out, err)
    }
    return nil
}

func (d *dnfManager) Remove(ctx context.Context, name string) error {
    cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", name)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("dnf remove %s: %s: %w", name, out, err)
    }
    return nil
}
```

### 2. Register in the factory

In `extensions/pkg/pkg.go`, add dnf to the detection logic:

```go
func detectManager(name string) PackageManager {
    switch name {
    case "apt":
        return &aptManager{}
    case "dnf":
        return &dnfManager{}
    // ...
    }
}
```

### 3. Add tests

Create `extensions/pkg/dnf_test.go` with table-driven tests:

```go
func TestDnfManager_Name(t *testing.T) {
    m := &dnfManager{}
    if m.Name() != "dnf" {
        t.Errorf("Name() = %q, want 'dnf'", m.Name())
    }
}
```

### 4. Open a PR

- Ensure `go test ./... -race` passes
- Ensure `go vet ./...` passes
- One file + one test file + factory registration
- No changes to `internal/` required

---

## Sub-Interfaces

Some extensions have sub-interfaces for platform-specific implementations:

| Extension | Sub-Interface | Implementations |
|-----------|--------------|-----------------|
| `pkg/` | `PackageManager` | apt, brew, choco, dnf, yum, zypper, apk, pacman, winget |
| `service/` | Platform build tags | systemd (Linux), launchd (macOS), SCM (Windows) |
| `firewall/` | Platform build tags | nftables/netlink (Linux), pf/anchor (macOS), registry API (Windows) |

To add a new package manager or init system, implement the sub-interface and register it. The engine doesn't change.

---

## Directory Structure

Each extension lives in its own subdirectory under `extensions/`. The shared `extension.go` and `state.go` define the interfaces and types: don't modify these. Current extensions: file, exec, firewall, pkg, service, user, registry, secpol, auditpol, sysctl, plist.

---

## Platform-Specific Extensions (Build Tags)

Use Go build tags to split platform-specific code. There are no stubs: if a platform doesn't need an extension, the DSL simply doesn't expose it.

### Extension layer: shared struct + build-tagged Check/Apply

Each extension has a shared file (no build tag) with the struct, `New()`, `ID()`, `String()`, `IsCritical()`. Platform-specific `Check()` and `Apply()` go in build-tagged files (e.g., `service_linux.go`, `service_windows.go`).

**Rules:**
1. The struct definition and `New()` constructor stay in the shared file (no build tag)
2. `Check()` and `Apply()` go in build-tagged files (one per platform)
3. Helper functions used only by one platform go in that platform's file
4. Windows extensions should use native Win32 APIs (via `golang.org/x/sys/windows` or `windows.NewLazySystemDLL`), not shell out to executables

### DSL layer: build-tagged methods and factories

If your extension is platform-specific (like Registry or Sysctl), you also need to wire it into the DSL. Cross-platform methods go in `dsl/run.go`, platform-specific methods in `dsl/run_<platform>.go`, and factories in `dsl/resources.go` or `dsl/resources_<platform>.go`.

**To add a platform-specific DSL method:**

1. Add the `Opts` struct to `dsl/dsl.go` (no build tag -- it's just a data type)
2. Add the `r.MyResource()` method to the appropriate `dsl/run_<platform>.go`
3. Add the `newMyResourceExtension()` factory to the appropriate `dsl/resources_<platform>.go`
4. Import your extension package in the factory file

The compiler enforces correctness: a Linux blueprint can call `r.Sysctl()` but not `r.Registry()`. No runtime "skipped" messages, no stubs.

### Blueprint layer: build-tagged files

If a blueprint calls platform-specific DSL methods, the blueprint file itself needs a build tag:

```go
//go:build windows

package cis

import "github.com/TsekNet/converge/dsl"

func WindowsCIS(r *dsl.Run) {
    r.Registry(`HKLM\...`, dsl.RegistryOpts{...})
}
```

### Testing platform-specific code

```go
func TestUser_Apply(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("unix-only test")
    }
    // test useradd logic
}
```

---

## Tips

- Keep extensions stateless -- all state comes from Check()
- Use `context.Context` for cancellation and timeouts
- Wrap errors with `fmt.Errorf("...: %w", err)` for debugging
- For Windows: prefer `golang.org/x/sys/windows/registry`, `golang.org/x/sys/windows/svc/mgr`, and `windows.NewLazySystemDLL` over `exec.Command`
- For Linux: prefer direct file I/O (`/proc/sys/`, `/etc/sysctl.d/`) over shelling out
- For macOS: prefer `howett.net/plist` for plist files over the `defaults` command
