# CLI Reference

Command-line interface for the Converge configuration management tool.

---

## Commands

### converge plan

Show what would change without modifying the system.

```
converge plan <blueprint>
```

Runs `Check()` on every resource and prints a grouped diff. Does not require root.

### converge apply

Apply changes to reach desired state.

```
converge apply <blueprint>
```

Runs `Check()` then `Apply()` on out-of-sync resources. Requires root (exit 10 if not).

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

```
converge v0.0.2
  commit: abc1234
  built:  2026-03-08T00:00:00Z
  go:     go1.26.0
  os:     linux/amd64
```

---

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--out` | | `terminal` | Output format (see below) |
| `--verbose` | `-v` | `false` | Show deck log output on stderr (also logged to syslog/eventlog) |
| `--timeout` | | `5m` | Per-resource timeout for Check/Apply cycles |
| `--parallel` | | `1` | Max concurrent resources (1 = sequential) |
| `--detailed-exit-codes` | | `false` | Use granular exit codes (2=changed, 3=partial, 4=all failed, 5=pending) |

### Output Formats

| Value | Description |
|-------|-------------|
| `terminal` | Unicode symbols, ANSI color, animated spinners, progress counter. Default. |
| `serial` | ASCII-only, no color, no escape codes, no spinners. For serial consoles, GCP, CI. |
| `json` | JSON object with full change details per resource. Machine-readable. |

---

## Exit Codes

By default, converge exits 0 on success (including changes applied and plan pending) and 1 on any failure. Pass `--detailed-exit-codes` for granular codes:

| Code | Meaning |
|------|---------|
| 0 | Success -- system already converged |
| 1 | General error (bad arguments, invalid blueprint, runtime failure) |
| 2 | Changes applied successfully (only with `--detailed-exit-codes`) |
| 3 | Partial failure (some resources failed, others applied) (only with `--detailed-exit-codes`) |
| 4 | All resources failed (only with `--detailed-exit-codes`) |
| 5 | Plan has pending changes (system not converged) (only with `--detailed-exit-codes`) |
| 10 | Permission denied (needs root/admin) |
| 11 | Blueprint not found |
| 12 | Platform not supported |

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

# Apply (requires root)
sudo converge apply workstation

# JSON output for CI scripting
converge plan workstation --out=json | jq '.resources[] | select(.status == "pending")'

# Serial mode for GCP/serial consoles
converge plan workstation --out=serial

# Parallel with timeout
sudo converge apply workstation --parallel=4 --timeout=2m

# Verbose (shows deck logs on stderr)
converge plan workstation -v

# List blueprints
converge list
converge list -b

# List extensions
converge list -e

# CIS hardening (platform-specific)
# CIS hardening (same command on every platform)
converge plan cis
sudo converge apply cis
```
