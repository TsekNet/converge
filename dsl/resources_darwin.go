//go:build darwin

package dsl

import (
	"github.com/TsekNet/converge/extensions"
	extplist "github.com/TsekNet/converge/extensions/plist"
)

func newPlistExtension(domain string, opts PlistOpts) extensions.Extension {
	p := extplist.New(domain, opts.Key)
	p.Value = opts.Value
	p.Type = opts.Type
	p.Host = opts.Host
	p.Critical = opts.Critical
	return p
}
