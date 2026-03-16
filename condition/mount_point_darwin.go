//go:build darwin

package condition

import (
	"context"

	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

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
