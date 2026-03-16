// Package exit defines all exit codes used by converge.
// These are the source of truth: no magic numbers elsewhere.
package exit

const (
	OK          = 0  // all resources in sync
	Error       = 1  // general error (check failure, build error)
	Changed     = 2  // one or more resources were changed
	PartialFail = 3  // some resources failed (including critical)
	AllFailed   = 4  // all resources failed
	Pending     = 5  // plan mode: changes pending
	NotRoot     = 10 // requires root/administrator
	NotFound    = 11 // blueprint not found
)
