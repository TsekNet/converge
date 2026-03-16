//go:build windows

package service

import (
	"time"

	"github.com/TsekNet/converge/extensions"
)

// PollInterval returns the polling interval for services on Windows.
// NotifyServiceStatusChangeW uses APCs which are incompatible with Go's
// goroutine scheduler (APCs are delivered to the calling OS thread, but
// goroutines migrate across threads). Polling is the reliable approach.
func (s *Service) PollInterval() time.Duration {
	return 5 * time.Second
}

var _ extensions.Poller = (*Service)(nil)
