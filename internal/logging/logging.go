package logging

import (
	"os"

	"github.com/TsekNet/converge/internal/version"
	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
)

// AppID re-exports the app name for platform-specific log init files.
var AppID = version.App

// Init sets up deck logging. Console (stderr) logging only appears in verbose mode
// to avoid polluting the pretty terminal output. Syslog/eventlog backends are
// always active via platform-specific init files.
func Init(verbose bool) {
	if verbose {
		deck.Add(logger.Init(os.Stderr, 0))
		deck.SetVerbosity(2)
	}
}
