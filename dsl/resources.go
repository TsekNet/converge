package dsl

import (
	"github.com/TsekNet/converge/extensions"
	extexec "github.com/TsekNet/converge/extensions/exec"
	extfile "github.com/TsekNet/converge/extensions/file"
	extpkg "github.com/TsekNet/converge/extensions/pkg"
	extreg "github.com/TsekNet/converge/extensions/registry"
	extsvc "github.com/TsekNet/converge/extensions/service"
	extuser "github.com/TsekNet/converge/extensions/user"
)

func newFileExtension(path string, opts FileOpts) extensions.Extension {
	f := extfile.New(path, opts.Content, opts.Mode)
	f.Owner = opts.Owner
	f.Group = opts.Group
	f.Append = opts.Append
	f.Critical = opts.Critical
	return f
}

func newPackageExtension(name string, opts PackageOpts, pkgManager string) extensions.Extension {
	p := extpkg.New(name, string(opts.State), pkgManager)
	p.Critical = opts.Critical
	return p
}

func newServiceExtension(name string, opts ServiceOpts, initSystem string) extensions.Extension {
	s := extsvc.New(name, string(opts.State), opts.Enable, initSystem)
	s.Critical = opts.Critical
	return s
}

func newExecExtension(name string, opts ExecOpts) extensions.Extension {
	e := extexec.New(name, opts.Command, opts.Args...)
	e.OnlyIf = opts.OnlyIf
	e.Dir = opts.Dir
	e.Env = opts.Env
	e.Retries = opts.Retries
	e.RetryDelay = opts.RetryDelay
	e.Critical = opts.Critical
	return e
}

func newUserExtension(name string, opts UserOpts) extensions.Extension {
	u := extuser.New(name, opts.Groups, opts.Shell)
	u.Home = opts.Home
	u.System = opts.System
	u.Critical = opts.Critical
	return u
}

func newRegistryExtension(key string, opts RegistryOpts) extensions.Extension {
	r := extreg.New(key)
	r.Value = opts.Value
	r.Type = opts.Type
	r.Data = opts.Data
	r.Critical = opts.Critical
	return r
}
