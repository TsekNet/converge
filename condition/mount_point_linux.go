//go:build linux

package condition

import (
	"context"
	"path/filepath"
	"syscall"

	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

type mountPointCondition struct {
	path string
}

// Met returns true when path is on a different device than its parent,
// indicating it is a mount point.
func (c *mountPointCondition) Met(_ context.Context) (bool, error) {
	var stat, parentStat syscall.Stat_t
	if err := syscall.Stat(c.path, &stat); err != nil {
		return false, nil //nolint:nilerr // not present = not met
	}
	parent := filepath.Dir(c.path)
	if err := syscall.Stat(parent, &parentStat); err != nil {
		return false, err
	}
	return stat.Dev != parentStat.Dev, nil
}

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

func (c *mountPointCondition) String() string {
	return "mount point " + c.path
}
