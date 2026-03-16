//go:build linux

package condition

import (
	"context"

	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Wait watches /proc/self/mountinfo via inotify. The kernel updates this file
// on every mount/unmount, making it the ideal trigger without requiring
// CAP_SYS_ADMIN or privileged netlink.
func (c *mountPointCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}

	w, err := watch.Shared()
	if err != nil {
		return err
	}

	const mountinfo = "/proc/self/mountinfo"
	ch, err := w.Watch(mountinfo, unix.IN_MODIFY)
	if err != nil {
		return err
	}
	defer w.Unwatch(mountinfo, ch)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-ch:
			if !ok {
				return ctx.Err()
			}
			if met, _ := c.Met(ctx); met {
				return nil
			}
		}
	}
}
