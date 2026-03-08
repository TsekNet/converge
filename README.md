<div align="center">
  <img src="assets/converge-banner-dark-gopher.png" alt="converge logo" width="400"/>
  <h1>converge</h1>
  <p><strong>Desired-state configuration, compiled.</strong> One binary. Every platform. Zero runtime deps.</p>

  [![codecov](https://codecov.io/gh/TsekNet/converge/branch/main/graph/badge.svg)](https://codecov.io/gh/TsekNet/converge)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
  [![GitHub Release](https://img.shields.io/github/v/release/TsekNet/converge)](https://github.com/TsekNet/converge/releases)
</div>

---

<div align="center">

![converge plan](assets/demo.gif)

</div>

*converge* compiles Linux, macOS, and Windows configurations into a single static binary. Write blueprints in Go, get compile-time type safety, and ship a ~3.5 MB executable with no interpreters, agents, or runtime dependencies.

> **Disclaimer:** This was created as a fun side project (PoC), not affiliated with any company.

## Install

Download latest installer for your platform from the [Releases](https://github.com/TsekNet/converge/releases) page.

## Quick start

**1. Edit a blueprint** (`blueprints/workstation.go`):

```go
package blueprints

import "github.com/TsekNet/converge/dsl"

func Workstation(r *dsl.Run) {
    r.Package("nginx", dsl.PackageOpts{State: dsl.Present})
    r.Service("nginx", dsl.ServiceOpts{State: dsl.Running, Enable: true})
    r.File("/etc/nginx/conf.d/app.conf", dsl.FileOpts{Content: "...", Mode: 0644})
}
```

**2. Plan and apply:**

```bash
converge plan workstation              # dry-run, no root needed
sudo converge apply workstation        # converge to desired state
```

**3. Flags:**

```bash
converge plan my-server --out=json             # machine-readable output (also: serial)
converge apply my-server --parallel 4          # run resources concurrently
converge apply my-server --timeout 2m          # per-resource timeout
converge plan my-server --detailed-exit-codes  # granular exit codes for CI
```

## Features

| Feature | Description |
|---------|-------------|
| **Compiled blueprints** | Go code: catch misconfigurations at build time, not at 2 AM |
| **Zero dependencies** | Single static binary, no Ruby/Python/JVM runtime |
| **Cross-platform** | Linux, macOS, Windows from one codebase with build tags |
| **Native OS APIs** | Win32 registry/SCM/LSA, Linux sysctl via `/proc/sys`, macOS plist via `howett.net/plist` -- no shelling out |
| **CIS benchmarks** | Built-in CIS L1 blueprints for [Windows 11](blueprints/cis/cis_windows.go), [Ubuntu 24.04](blueprints/cis/cis_linux.go), and [macOS 15](blueprints/cis/cis_darwin.go) |
| **Plan / Apply** | Dry-run any blueprint before making changes |
| **Parallel execution** | Concurrent resource application with configurable parallelism |
| **Extensible** | Implement the `Extension` interface to add new resource types |

## Why converge?

| | Converge | Chef | Puppet | Ansible | Terraform |
|-|----------|------|--------|---------|-----------|
| **Language** | Go | Ruby | Ruby DSL | YAML | HCL |
| **Runtime deps** | None | Ruby | JVM | Python | None |
| **Config format** | Go code | Ruby DSL | Ruby DSL | YAML | HCL |
| **Type safety** | Compile-time | Runtime | Runtime | Runtime | Runtime |
| **Binary size** | ~3.5 MB | ~600 MB | ~44 MB | ~500 MB | ~96 MB |
| **State file** | No | No | No | No | Yes |
| **IDE support** | Full Go tooling | Limited | Limited | YAML only | Limited |

## Documentation

| Doc | Description |
|-----|-------------|
| [Design](docs/design.md) | Philosophy, architecture, engine flow, native API strategy |
| [Guide](docs/guide.md) | Blueprint writing, composition, testing, full resource reference |
| [CLI](docs/cli.md) | Commands, flags, exit codes, output formats |
| [Extensions](docs/extending.md) | Adding new extensions and platform-specific resources |
| [Blueprints](blueprints/) | Built-in blueprints including [CIS benchmarks](blueprints/cis/) |

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

See [CONTRIBUTING.md](CONTRIBUTING.md) for dev setup, code standards, and PR checklist.

## License

[MIT](LICENSE)
