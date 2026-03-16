//go:build darwin

package user

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for user account state.
// On Windows, WMI event subscription is used instead (see watch_windows.go).
func (u *User) PollInterval() time.Duration {
	return 60 * time.Second
}

var _ extensions.Poller = (*User)(nil)
