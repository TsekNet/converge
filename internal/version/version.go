package version

// Name is the application name. Single source of truth for CLI, logging,
// firewall rules, and service registration.
const Name = "converge"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
