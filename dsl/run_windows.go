//go:build windows

package dsl

func (r *Run) Registry(key string, opts RegistryOpts) {
	mustNotBeEmpty("Registry", "key", key)
	r.addResource(newRegistryExtension(key, opts), opts.DependsOn)
}

func (r *Run) SecurityPolicy(name string, opts SecurityPolicyOpts) {
	mustNotBeEmpty("SecurityPolicy", "name", name)
	mustNotBeEmpty("SecurityPolicy", "category", opts.Category)
	mustNotBeEmpty("SecurityPolicy", "key", opts.Key)
	r.addResource(newSecurityPolicyExtension(name, opts), opts.DependsOn)
}

func (r *Run) AuditPolicy(name string, opts AuditPolicyOpts) {
	mustNotBeEmpty("AuditPolicy", "name", name)
	mustNotBeEmpty("AuditPolicy", "subcategory", opts.Subcategory)
	r.addResource(newAuditPolicyExtension(name, opts), opts.DependsOn)
}
