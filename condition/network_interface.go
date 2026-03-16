package condition

import (
	"context"
	"net"
)

// networkInterfaceCondition is satisfied when the named interface exists and is up.
// Wait() is platform-specific (netlink on Linux, NotifyIpInterfaceChange on Windows,
// 2s poll on macOS).
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

func (c *networkInterfaceCondition) String() string {
	return "network interface " + c.name + " up"
}
