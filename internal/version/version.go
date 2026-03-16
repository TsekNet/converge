package version

// App is the application name. Single source of truth for CLI, logging,
// firewall rules, and service registration.
const App = "converge"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
