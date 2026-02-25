# Adding a New Extension

**[← Wiki Home](Home)** · [Design](Design) · [Guide](Guide) · [CLI](CLI)

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
| `service/` | `ServiceManager` | systemd, launchd, windows, (add: openrc, runit) |

To add a new package manager or init system, implement the sub-interface and register it. The engine doesn't change.

---

## Directory Structure

```
extensions/
├── extension.go          # Extension interface (don't modify)
├── state.go              # State/Change/Result types (don't modify)
├── file/                 # File management
├── exec/                 # Command execution
├── pkg/                  # Package management (add new managers here)
├── service/              # Service management (add new init systems here)
├── user/                 # User/group management
└── registry/             # Windows registry
```

---

## Platform-Specific Extensions (Build Tags)

Use Go build tags to split platform-specific code. The pattern:

```
extensions/user/
├── user.go            # Shared: struct, New(), ID(), String(), Check()
├── user_linux.go      # //go:build linux  -- Apply() using useradd/usermod
├── user_darwin.go     # //go:build darwin -- Apply() using dscl
├── user_windows.go    # //go:build windows  -- Apply() using net user
└── user_test.go       # Tests (Check tests run everywhere, Apply tests skip wrong platform)
```

**Rules:**
1. `Check()` should be cross-platform (reading state is usually portable)
2. `Apply()` goes in build-tagged files (one per platform)
3. Helper functions used only by one platform go in that platform's file
4. The struct definition and `New()` constructor stay in the shared file

**Example: no-op stub for unsupported platform**

If an extension only makes sense on one OS (e.g., Windows Registry), provide a no-op stub:

```go
// registry_stub.go
//go:build linux || darwin

func (r *Registry) Check(_ context.Context) (*extensions.State, error) {
    return &extensions.State{InSync: true}, nil  // always "in sync" = skip
}

func (r *Registry) Apply(_ context.Context) (*extensions.Result, error) {
    return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "Skipped (not Windows)"}, nil
}
```

**Testing platform-specific code:**

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
- Put `Check()` in the shared file, `Apply()` in build-tagged files
- No-op stubs should return `InSync: true` (skip) not `InSync: false` (false alarm)
