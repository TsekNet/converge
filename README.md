<div align="center">
  <img src="assets/converge-banner-dark-gopher.png" alt="converge logo" width="400"/>
  <h1>Converge</h1>
  <p><strong>Desired State Configuration, Compiled.</strong></p>

  [![codecov](https://codecov.io/gh/TsekNet/converge/branch/main/graph/badge.svg)](https://codecov.io/gh/TsekNet/converge)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
  [![GitHub Release](https://img.shields.io/github/v/release/TsekNet/converge)](https://github.com/TsekNet/converge/releases)
</div>

---

Compile all your linux/mac/windows configurations into a single binary. Inspired by tools like Chef, Puppet, Ansible, and Terraform.

<div align="center">

![converge plan](assets/demo.gif)

</div>

> **Disclaimer:** This was created as a fun side project (PoC), not affiliated with any company.

### Why Converge?

| Feature | Converge | Chef | Puppet | Ansible | Terraform |
|---------|----------|------|--------|---------|-----------|
| Language | Go | Ruby | Ruby DSL | YAML | HCL |
| Runtime deps | None | Ruby | JVM | Python | None |
| Config format | Go code | Ruby DSL | Ruby DSL | YAML | HCL |
| Type safety | Compile-time | Runtime | Runtime | Runtime | Runtime |
| Binary size | ~4 MB | Large install | Large install | Python + deps | ~80 MB |
| State file | No | No | No | No | Yes |
| IDE support | Full Go tooling | Limited | Limited | YAML only | Limited |

## Install

To play around with converge before writing your first blueprint, grab a binary from [Releases](https://github.com/TsekNet/converge/releases).

## Quick Start

**1. Modify a blueprint** (`blueprints/workstation.go`):

```go
package blueprints

import "github.com/TsekNet/converge/dsl"

func Workstation(r *dsl.Run) {
    r.Package("nginx", dsl.PackageOpts{State: dsl.Present})
    r.Service("nginx", dsl.ServiceOpts{State: dsl.Running, Enable: true})
    r.File("/etc/nginx/conf.d/app.conf", dsl.FileOpts{Content: "...", Mode: 0644})
}
```

**2. Build, plan, apply:**

```bash
go build -o converge ./cmd/converge
converge plan workstation              # dry-run, no root needed
sudo converge apply workstation        # converge to desired state
```

### Flags

```bash
converge plan my-server --out=json           # machine-readable output (also: serial)
converge apply my-server --parallel 4        # run resources concurrently
converge apply my-server --timeout 2m        # per-resource timeout
converge plan my-server --detailed-exit-codes  # granular exit codes for CI
```

## Documentation

**[📚 Wiki](https://github.com/TsekNet/converge/wiki)** — Design, Guide, CLI, Extending. Wiki source lives in `wiki/` in this repo.

Built-in blueprints: `converge list` shows available blueprints (`workstation`, `linux`, `darwin`, `windows`, `linux_server`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for dev setup, code standards, and PR checklist.
