package auditpol

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for audit policy state.
func (a *AuditPolicy) PollInterval() time.Duration {
	return 60 * time.Second
}

var _ extensions.Poller = (*AuditPolicy)(nil)
