//go:build linux

package condition

import (
	"context"
	"path/filepath"

	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Wait uses the shared inotify multiplexer to watch the parent directory for
// IN_CREATE|IN_MOVED_TO events, then re-checks Met on each notification.
func (c *fileExistsCondition) Wait(ctx context.Context) error {
	w, err := watch.Shared()
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.path)
	ch, err := w.Watch(dir, unix.IN_CREATE|unix.IN_MOVED_TO)
	if err != nil {
		return err
	}
	defer w.Unwatch(dir, ch)

	// Check immediately in case the file appeared before we subscribed.
	if met, _ := c.Met(ctx); met {
		return nil
	}

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
