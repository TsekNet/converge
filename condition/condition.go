// Package condition provides extensions.Condition implementations that gate
// resource convergence on system state. All implementations use OS-native APIs
// or pure-Go net/os calls. No exec.Command.
package condition

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

// FileExists returns a Condition that is satisfied when os.Stat(path) succeeds.
// Wait uses the shared OS file watcher (inotify/kqueue) to avoid polling.
func FileExists(path string) *fileExistsCondition {
	return &fileExistsCondition{path: path}
}

// NetworkReachable returns a Condition satisfied when a TCP connection to
// host:port succeeds. Wait uses a 5-second poll internally: TCP reachability
// has no kernel event API that is portable across Linux/macOS/Windows without
// CGO. This is the justified exception to the "no polling" rule.
func NetworkReachable(host string, port int) *networkReachableCondition {
	return &networkReachableCondition{host: host, port: port}
}

// NetworkInterface returns a Condition satisfied when the named network
// interface exists and is up. Wait uses OS-native event APIs (netlink on
// Linux, NotifyIpInterfaceChange on Windows, 2s poll on macOS).
func NetworkInterface(name string) *networkInterfaceCondition {
	return &networkInterfaceCondition{name: name}
}

// MountPoint returns a Condition satisfied when path is a mount point
// (on a different device than its parent directory). Wait uses inotify on
// /proc/self/mountinfo on Linux, kqueue on macOS, and polling on Windows.
func MountPoint(path string) *mountPointCondition {
	return &mountPointCondition{path: path}
}

// networkReachableCondition polls TCP connectivity. Cross-platform.
type networkReachableCondition struct {
	host string
	port int
}

func (c *networkReachableCondition) Met(_ context.Context) (bool, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), 2*time.Second)
	if err != nil {
		return false, nil //nolint:nilerr // unreachable is not an error, just not met
	}
	conn.Close()
	return true, nil
}

func (c *networkReachableCondition) Wait(ctx context.Context) error {
	// No kernel event for TCP reachability without CGO. 5s poll is the
	// pragmatic cross-platform approach, documented here intentionally.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if met, _ := c.Met(ctx); met {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *networkReachableCondition) String() string {
	return fmt.Sprintf("network reachable %s:%d", c.host, c.port)
}

// fileExistsCondition checks for path existence. Wait impl is platform-specific.
type fileExistsCondition struct {
	path string
}

func (c *fileExistsCondition) Met(_ context.Context) (bool, error) {
	_, err := os.Stat(c.path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (c *fileExistsCondition) String() string {
	return fmt.Sprintf("file exists %s", c.path)
}
