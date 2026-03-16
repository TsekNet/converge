package user

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for user account state.
// No OS provides native events for user account changes.
func (u *User) PollInterval() time.Duration {
	return 60 * time.Second
}

var _ extensions.Poller = (*User)(nil)
