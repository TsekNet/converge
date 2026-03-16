package firewall

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for firewall rules.
// No platform provides reliable native events for firewall rule changes.
func (fw *Firewall) PollInterval() time.Duration {
	return 30 * time.Second
}

var _ extensions.Poller = (*Firewall)(nil)
