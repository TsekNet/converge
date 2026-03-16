//go:build darwin

package service

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for services on macOS.
// launchd has no native notification API, so we poll.
func (s *Service) PollInterval() time.Duration {
	return 10 * time.Second
}

// Ensure Service implements Poller at compile time.
var _ extensions.Poller = (*Service)(nil)
