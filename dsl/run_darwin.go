//go:build darwin

package dsl

func (r *Run) Plist(domain string, opts PlistOpts) {
	if err := requireNotEmpty("Plist", "domain", domain); err != nil {
		r.err = err
		return
	}
	if err := requireNotEmpty("Plist", "key", opts.Key); err != nil {
		r.err = err
		return
	}
	r.addResource(newPlistExtension(domain, opts), opts.DependsOn)
}
