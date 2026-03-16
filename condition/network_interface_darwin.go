//go:build darwin

package condition

import (
	"context"
	"time"
)

// Wait polls at 2-second intervals. macOS SCNetworkReachability callbacks
// require a CoreFoundation CFRunLoop and are not accessible from pure Go
// without CGO. This is the documented limitation for macOS.
func (c *networkInterfaceCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if met, _ := c.Met(ctx); met {
				return nil
			}
		}
	}
}
