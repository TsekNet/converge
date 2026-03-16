//go:build windows

package dsl

func (r *Run) Registry(key string, opts RegistryOpts) {
	if err := requireNotEmpty("Registry", "key", key); err != nil {
		r.err = err
		return
	}
	r.addResource(newRegistryExtension(key, opts), opts.DependsOn)
}

func (r *Run) SecurityPolicy(name string, opts SecurityPolicyOpts) {
	if err := requireNotEmpty("SecurityPolicy", "name", name); err != nil {
		r.err = err
		return
	}
	if err := requireNotEmpty("SecurityPolicy", "category", opts.Category); err != nil {
		r.err = err
		return
	}
	if err := requireNotEmpty("SecurityPolicy", "key", opts.Key); err != nil {
		r.err = err
		return
	}
	r.addResource(newSecurityPolicyExtension(name, opts), opts.DependsOn)
}

func (r *Run) AuditPolicy(name string, opts AuditPolicyOpts) {
	if err := requireNotEmpty("AuditPolicy", "name", name); err != nil {
		r.err = err
		return
	}
	if err := requireNotEmpty("AuditPolicy", "subcategory", opts.Subcategory); err != nil {
		r.err = err
		return
	}
	r.addResource(newAuditPolicyExtension(name, opts), opts.DependsOn)
}
