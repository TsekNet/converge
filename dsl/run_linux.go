//go:build linux

package dsl

func (r *Run) Sysctl(key string, opts SysctlOpts) {
	if err := requireNotEmpty("Sysctl", "key", key); err != nil {
		r.err = err
		return
	}
	if err := requireNotEmpty("Sysctl", "value", opts.Value); err != nil {
		r.err = err
		return
	}
	r.addResource(newSysctlExtension(key, opts), opts.DependsOn)
}
