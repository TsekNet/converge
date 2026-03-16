package secpol

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for security policy state.
func (sp *SecurityPolicy) PollInterval() time.Duration {
	return 60 * time.Second
}

var _ extensions.Poller = (*SecurityPolicy)(nil)
