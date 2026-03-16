package dsl

import (
	"github.com/TsekNet/converge/extensions"
	extexec "github.com/TsekNet/converge/extensions/exec"
	extfile "github.com/TsekNet/converge/extensions/file"
	extfw "github.com/TsekNet/converge/extensions/firewall"
	extpkg "github.com/TsekNet/converge/extensions/pkg"
	extsvc "github.com/TsekNet/converge/extensions/service"
	extuser "github.com/TsekNet/converge/extensions/user"
)

func newFileExtension(path string, opts FileOpts) extensions.Extension {
	f := extfile.New(path, opts.Content, opts.Mode)
	f.Owner = opts.Owner
	f.Group = opts.Group
	f.Append = opts.Append
	f.Critical = opts.Meta.Critical
	return f
}

func newPackageExtension(name string, opts PackageOpts, pkgManager string) extensions.Extension {
	p := extpkg.New(name, string(opts.State), pkgManager)
	p.Critical = opts.Meta.Critical
	return p
}

func newServiceExtension(name string, opts ServiceOpts, initSystem string) extensions.Extension {
	s := extsvc.New(name, string(opts.State), opts.Enable, initSystem)
	s.StartupType = opts.StartupType
	s.Critical = opts.Meta.Critical
	return s
}

func newExecExtension(name string, opts ExecOpts) extensions.Extension {
	e := extexec.New(name, opts.Command, opts.Args...)
	e.OnlyIf = opts.OnlyIf
	e.Dir = opts.Dir
	e.Env = opts.Env
	e.Retries = opts.Retries
	e.RetryDelay = opts.RetryDelay
	e.Critical = opts.Meta.Critical
	return e
}

func newUserExtension(name string, opts UserOpts) extensions.Extension {
	u := extuser.New(name, opts.Groups, opts.Shell)
	u.Home = opts.Home
	u.System = opts.System
	u.Critical = opts.Meta.Critical
	return u
}

func newFirewallExtension(name string, opts FirewallOpts) extensions.Extension {
	f := extfw.New(name, opts.Port, opts.Protocol, opts.Direction, opts.Action)
	f.Source = opts.Source
	f.Dest = opts.Dest
	f.Critical = opts.Meta.Critical
	if opts.State == Absent {
		f.State = "absent"
	}
	return f
}
