//go:build linux

package dsl

func (r *Run) Sysctl(key string, opts SysctlOpts) {
	mustNotBeEmpty("Sysctl", "key", key)
	mustNotBeEmpty("Sysctl", "value", opts.Value)
	r.addResource(newSysctlExtension(key, opts))
}
