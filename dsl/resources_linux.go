//go:build linux

package dsl

import (
	"github.com/TsekNet/converge/extensions"
	extsysctl "github.com/TsekNet/converge/extensions/sysctl"
)

func newSysctlExtension(key string, opts SysctlOpts) extensions.Extension {
	s := extsysctl.New(key, opts.Value)
	s.Persist = opts.Persist
	s.Critical = opts.Meta.Critical
	return s
}
