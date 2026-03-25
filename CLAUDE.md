# Converge

Event-driven configuration management daemon. Single static binary, zero runtime deps. Detects and fixes drift via OS-level events (inotify, dbus, Win32), not cron.

## Architecture

Read `docs/design.md` for philosophy and security model, `docs/extensions.md` for adding resources, `docs/examples.md` for blueprint authoring, `docs/cli.md` for commands and exit codes.

Key layers:

| Layer | Visibility | Location |
| --- | --- | --- |
| DSL (blueprint API) | Public | `dsl/` (platform-specific methods in build-tagged files) |
| Extensions (OS resources) | Public | `extensions/` (file, pkg, service, exec, user, firewall, registry, secpol, auditpol, sysctl, plist) |
| Blueprints | Public | `blueprints/` (baseline, per-OS, CIS benchmarks) |
| Engine + internals | Private | `internal/` (engine, graph, daemon, watch, output, platform, logging, exit) |
| CLI | Binary | `cmd/converge/` |

## Build and test

```bash
go build -o bin/converge ./cmd/converge
go test -race ./...
go vet ./...
```

Linux integration tests: `sudo bash .github/ci/scripts/test-linux.sh`

## Code standards

From `CONTRIBUTING.md`: Go 1.26+, table-driven tests, build tags `_linux`/`_darwin`/`_windows` (not `_unix` or `!windows`), native OS APIs (no shell-outs), error wrapping with `%w`, logging via `google/deck`, builds via GoReleaser.

## Platform-specific code

Build-tagged files use `_linux.go`, `_darwin.go`, `_windows.go` suffixes AND `//go:build` directives. No `_unix.go` files. Platform-specific DSL methods only exist in the build-tagged file for that platform (no stubs).

Extensions with OS-specific implementations: `extensions/pkg/` (apt/brew/winget/pacman), `extensions/service/` (systemd/launchd/SCM), `extensions/watch/` (inotify/dbus/WinAPI), `extensions/firewall/` (nftables/pf/Windows Firewall).

## Extension interface

Every resource implements `Check(ctx) (*State, error)` and `Apply(ctx) (*Result, error)`. Optional: `Watch()` (OS events), `Poller` (periodic), `CriticalResource` (stop on failure). See `extensions/extension.go`.

## Agents

Three agents in `.claude/agents/`. Use when changes touch their domain:

| Agent | Trigger files |
| --- | --- |
| `platform-reviewer` | `_windows.go`, `_darwin.go`, `_linux.go`, `build/`, `runtime.GOOS` branches, `internal/platform/` |
| `security-auditor` | `extensions/exec/`, `dsl/config.go` (secrets), `extensions/registry/`, `extensions/secpol/`, `extensions/auditpol/` |
| `extension-reviewer` | New or modified files in `extensions/`, `dsl/resources*.go`, `condition/` |
