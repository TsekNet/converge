//go:build linux

package dsl

func (r *Run) Sysctl(key string, opts SysctlOpts) {
	if !r.require("Sysctl", "key", key) {
		return
	}
	if !r.require("Sysctl", "value", opts.Value) {
		return
	}
	r.addResource(newSysctlExtension(key, opts), opts.Meta.DependsOn)
}
