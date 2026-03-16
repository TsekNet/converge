package exec

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for exec resources.
// No OS provides native events for arbitrary command state.
func (e *Exec) PollInterval() time.Duration {
	return 30 * time.Second
}

var _ extensions.Poller = (*Exec)(nil)
