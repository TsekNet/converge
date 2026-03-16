# CLI Reference

Command-line interface for the Converge configuration management tool.

---

## Commands

### converge serve

Run as a persistent daemon, watching for drift and re-converging immediately.

```
converge serve <blueprint> [flags]
```

Builds a DAG of all resources, performs initial convergence, then starts per-resource watchers. Resources with native OS event support (File via inotify, Service via dbus) detect drift instantly. Others poll at configurable intervals.

| Flag | Default | Description |
|------|---------|-------------|
| `--once` | `false` | Exit after initial convergence (CI/Packer mode) |
| `--max-retries` | `3` | Max retries before marking a resource noncompliant |

Requires root (exit 10 if not).

### converge plan

Show what would change without modifying the system.

```
converge plan <blueprint>
```

Runs `Check()` on every resource in topological order and prints a grouped diff. Does not require root.

### converge list

List registered blueprints and/or extensions.

```
converge list
converge list --blueprints
converge list --extensions
```

| Flag | Short | Description |
|------|-------|-------------|
| `--blueprints` | `-b` | Show only blueprints |
| `--extensions` | `-e` | Show only extensions |

Built-in blueprints vary by platform:

| Blueprint | Platform | Description |
|-----------|----------|-------------|
| `workstation` | All | Base workstation setup |
| `linux` | Linux | Linux-specific defaults |
| `linux_server` | Linux | Hardened Linux server |
| `darwin` | macOS | macOS-specific defaults |
| `windows` | Windows | Windows-specific defaults |
| `cis` | All | CIS L1 security benchmark (platform-specific) |

### converge version

Print build information.

```
converge version
```

---

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--out` | | `terminal` | Output format (see below) |
| `--verbose` | `-v` | `false` | Show deck log output on stderr (also logged to syslog/eventlog) |
| `--timeout` | | `5m` | Per-resource timeout for Check/Apply cycles |
| `--parallel` | | `1` | Max concurrent resources within each DAG layer (1 = sequential) |
| `--detailed-exit-codes` | | `false` | Use granular exit codes (see below) |

### Output Formats

| Value | Description |
|-------|-------------|
| `terminal` | Unicode symbols, ANSI color, animated spinners, progress counter. Default. |
| `serial` | ASCII-only, no color, no escape codes, no spinners. For serial consoles, GCP, CI. |
| `json` | JSON object with full change details per resource. Machine-readable. |

---

## Exit Codes

Defined in `internal/exit/exit.go`. By default, converge exits 0 on success and 1 on failure. Pass `--detailed-exit-codes` for granular codes:

| Code | Name | Meaning |
|------|------|---------|
| 0 | OK | System converged, no changes needed |
| 1 | Error | General error (bad arguments, invalid blueprint, runtime failure) |
| 2 | Changed | Changes applied successfully |
| 3 | PartialFail | Some resources failed, others applied |
| 4 | AllFailed | All resources failed |
| 5 | Pending | Plan mode: changes pending |
| 10 | NotRoot | Requires root/administrator |
| 11 | NotFound | Blueprint not found |

---

## Service Installation

Converge runs as a system service on all platforms. Packages install and start the service automatically.

### Linux (systemd)

```bash
sudo dpkg -i converge.deb           # installs, enables, and starts
sudo systemctl status converge       # check status
sudo journalctl -u converge -f       # follow logs
sudo systemctl restart converge      # restart after binary update
```

Service file: `/usr/lib/systemd/system/converge.service`

### macOS (launchd)

```bash
sudo installer -pkg converge.pkg -target /  # installs and starts
sudo launchctl list | grep converge          # check status
tail -f /var/log/converge.log                # follow logs
```

Plist: `/Library/LaunchDaemons/com.tseknet.converge.plist`

### Windows (SCM)

The MSI installer registers and starts the `converge` Windows service automatically.

```powershell
Get-Service converge                 # check status
Restart-Service converge             # restart after binary update
Get-EventLog -LogName Application -Source converge  # view logs
```

### Upgrades

Replace the binary via your package manager. The service manager restarts converge automatically:

| Platform | Upgrade command | Restart |
|---|---|---|
| Linux | `sudo dpkg -i converge.deb` | postinst runs `systemctl restart` |
| macOS | `sudo installer -pkg converge.pkg -target /` | postinstall runs `launchctl bootstrap` |
| Windows | Run new `converge.msi` | MSI stops/starts the service |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NO_COLOR` | Disables color output in terminal mode. Follows the [no-color standard](https://no-color.org/). |
| `CONVERGE_OUT` | Default output format. Overridden by `--out`. |

---

## Examples

```bash
# Plan (dry-run, no root)
converge plan workstation

# Serve as persistent daemon (requires root)
sudo converge serve workstation

# Converge once and exit (CI/Packer)
sudo converge serve workstation --once

# JSON output for CI scripting
converge plan workstation --out=json | jq '.resources[] | select(.status == "pending")'

# Parallel with timeout
sudo converge serve workstation --parallel=4 --timeout=2m

# Custom retry limit
sudo converge serve workstation --max-retries=5

# List blueprints
converge list -b

# CIS hardening
converge plan cis
sudo converge serve cis --once
```
