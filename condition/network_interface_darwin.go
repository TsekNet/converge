//go:build darwin

package condition

import (
	"context"
	"net"
	"time"
)

type networkInterfaceCondition struct {
	name string
}

func (c *networkInterfaceCondition) Met(_ context.Context) (bool, error) {
	iface, err := net.InterfaceByName(c.name)
	if err != nil {
		return false, nil //nolint:nilerr // not found = not met
	}
	return iface.Flags&net.FlagUp != 0, nil
}

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

func (c *networkInterfaceCondition) String() string {
	return "network interface " + c.name + " up"
}
