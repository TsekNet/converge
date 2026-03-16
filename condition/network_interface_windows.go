//go:build windows

package condition

import (
	"context"
	"net"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
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

// Wait uses NotifyIpInterfaceChange (iphlpapi) to receive a callback on any
// IP interface change, then re-checks Met. This avoids polling entirely on
// Windows.
func (c *networkInterfaceCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}

	var (
		mu      sync.Mutex
		notifyCh = make(chan struct{}, 1)
	)

	// The callback is called from a Windows thread pool thread.
	// It signals the Go channel; actual Met() check happens in the Go goroutine.
	cb := windows.NewCallback(func(callerCtx uintptr, row uintptr, notifType uint32) uintptr {
		mu.Lock()
		select {
		case notifyCh <- struct{}{}:
		default:
		}
		mu.Unlock()
		return 0
	})

	var handle windows.Handle
	err := notifyIpInterfaceChange(windows.AF_UNSPEC, cb, 0, false, &handle)
	if err != nil {
		return err
	}
	defer cancelMibChangeNotify2(handle)

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

func (c *networkInterfaceCondition) String() string {
	return "network interface " + c.name + " up"
}

// iphlpapi syscalls for interface change notifications.
var (
	modIphlpapi             = windows.NewLazySystemDLL("iphlpapi.dll")
	procNotifyIpIface       = modIphlpapi.NewProc("NotifyIpInterfaceChange")
	procCancelMibChangeNotify2 = modIphlpapi.NewProc("CancelMibChangeNotify2")
)

func notifyIpInterfaceChange(family uint16, callback uintptr, callerCtx uintptr, initialNotification bool, handle *windows.Handle) error {
	init := uintptr(0)
	if initialNotification {
		init = 1
	}
	r1, _, e := procNotifyIpIface.Call(
		uintptr(family),
		callback,
		callerCtx,
		init,
		uintptr(unsafe.Pointer(handle)),
	)
	if r1 != 0 {
		return e
	}
	return nil
}

func cancelMibChangeNotify2(handle windows.Handle) {
	procCancelMibChangeNotify2.Call(uintptr(handle))
}
