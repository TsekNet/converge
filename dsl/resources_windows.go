//go:build windows

package dsl

import (
	"github.com/TsekNet/converge/extensions"
	extaudit "github.com/TsekNet/converge/extensions/auditpol"
	extreg "github.com/TsekNet/converge/extensions/registry"
	extsecpol "github.com/TsekNet/converge/extensions/secpol"
)

func newRegistryExtension(key string, opts RegistryOpts) extensions.Extension {
	r := extreg.New(key)
	r.Value = opts.Value
	r.Type = opts.Type
	r.Data = opts.Data
	r.Critical = opts.Meta.Critical
	if opts.State == Absent {
		r.State = "absent"
	}
	return r
}

func newSecurityPolicyExtension(_ string, opts SecurityPolicyOpts) extensions.Extension {
	s := extsecpol.New(opts.Category, opts.Key, opts.Value)
	s.Critical = opts.Meta.Critical
	return s
}

func newAuditPolicyExtension(_ string, opts AuditPolicyOpts) extensions.Extension {
	a := extaudit.New(opts.Subcategory, opts.Success, opts.Failure)
	a.Critical = opts.Meta.Critical
	return a
}
