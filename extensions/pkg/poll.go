package pkg

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for package state.
// No OS provides native events for package installation changes.
func (p *Package) PollInterval() time.Duration {
	return 5 * time.Minute
}

var _ extensions.Poller = (*Package)(nil)
