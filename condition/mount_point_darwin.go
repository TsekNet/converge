//go:build darwin

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

// Wait watches the filesystem root via kqueue NOTE_WRITE. Any mount event
// causes a write to "/" (mtime update), making this an event-driven trigger
// without polling. It over-triggers on root writes but Met() filters correctly.
func (c *mountPointCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}

	w, err := watch.Shared()
	if err != nil {
		return err
	}

	ch, err := w.Watch("/", unix.NOTE_WRITE|unix.NOTE_ATTRIB)
	if err != nil {
		return err
	}
	defer w.Unwatch("/", ch)

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
