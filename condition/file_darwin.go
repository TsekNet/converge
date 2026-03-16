//go:build darwin

package condition

import (
	"context"
	"path/filepath"

	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Wait uses the shared kqueue multiplexer to watch the parent directory for
// NOTE_WRITE events, then re-checks Met on each notification.
func (c *fileExistsCondition) Wait(ctx context.Context) error {
	w, err := watch.Shared()
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.path)
	ch, err := w.Watch(dir, unix.NOTE_WRITE)
	if err != nil {
		return err
	}
	defer w.Unwatch(dir, ch)

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
