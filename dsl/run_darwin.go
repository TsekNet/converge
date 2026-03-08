//go:build darwin

package dsl

func (r *Run) Plist(domain string, opts PlistOpts) {
	mustNotBeEmpty("Plist", "domain", domain)
	mustNotBeEmpty("Plist", "key", opts.Key)
	r.addResource(newPlistExtension(domain, opts))
}
