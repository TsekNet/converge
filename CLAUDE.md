# Converge

Event-driven configuration management daemon. Single static binary, zero runtime deps. Detects and fixes drift via OS-level events (inotify, dbus, Win32), not cron.

## Architecture

Read `docs/design.md` for philosophy and security model, `docs/extensions.md` for adding resources, `docs/examples.md` for blueprint authoring, `docs/cli.md` for commands and exit codes.

| Layer | Location |
| --- | --- |
| DSL (blueprint API) | `dsl/` (platform-specific methods in build-tagged files) |
| Extensions | `extensions/` (exec, file, firewall, pkg, reboot, service, user, registry, secpol, auditpol, sysctl, plist) |
| Conditions | `condition/` (FileExists, MountPoint, NetworkInterface, NetworkReachable, RegistryKeyExists) |
| Blueprints | `blueprints/` (baseline, per-OS, CIS benchmarks) |
| Internals | `internal/` (daemon, engine, exit, graph, logging, output, platform, version, watch, winreg) |
| CLI | `cmd/converge/` |

## Build and test

```bash
go build -o bin/converge ./cmd/converge && go test -race ./... && go vet ./...
```

Integration tests: `sudo bash .github/ci/scripts/test-linux.sh`

## Code standards

See `CONTRIBUTING.md`. Key non-obvious rules: build tags use `_linux`/`_darwin`/`_windows` (never `_unix` or `!windows`), no shell-outs (native OS APIs only), `//go:build` directives required alongside filename suffixes.

## Extension interface

Defined in `extensions/extension.go`. Every resource implements `Check` (read-only) and `Apply` (mutate). Optional interfaces: `Watcher`, `Poller`, `CriticalResource`. Conditions implement `Met`, `Wait`, `String`.

## Agents

Three agents in `.claude/agents/`. Use when changes touch their domain:

| Agent | Trigger files |
| --- | --- |
| `platform-reviewer` | `_windows.go`, `_darwin.go`, `_linux.go`, `build/`, `runtime.GOOS` branches, `internal/platform/`, `condition/*_windows.go` |
| `security-auditor` | `extensions/exec/`, `dsl/config.go` (secrets), `extensions/registry/`, `extensions/secpol/`, `extensions/auditpol/`, `condition/` |
| `extension-reviewer` | New or modified files in `extensions/`, `dsl/resources*.go`, `condition/` |
