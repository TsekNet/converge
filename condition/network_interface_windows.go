//go:build windows

package condition

import (
	"context"

	"golang.org/x/sys/windows"
)

// Wait uses NotifyIpInterfaceChange (iphlpapi) to receive a callback on any
// IP interface change, then re-checks Met. This avoids polling entirely on
// Windows.
func (c *networkInterfaceCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}

	// notifyCh is buffered so the Windows thread pool callback never blocks.
	// No mutex needed: select/default on a buffered channel is goroutine-safe.
	notifyCh := make(chan struct{}, 1)

	cb := windows.NewCallback(func(_ uintptr, _ uintptr, _ uint32) uintptr {
		select {
		case notifyCh <- struct{}{}:
		default:
		}
		return 0
	})

	var handle windows.Handle
	if err := windows.NotifyIpInterfaceChange(windows.AF_UNSPEC, cb, nil, false, &handle); err != nil {
		return err
	}
	defer windows.CancelMibChangeNotify2(handle)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-notifyCh:
			if met, _ := c.Met(ctx); met {
				return nil
			}
		}
	}
}
