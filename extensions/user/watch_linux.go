//go:build linux

package user

import (
	"context"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/watch"
)

// Watch uses the shared inotify multiplexer to monitor /etc/passwd for
// user account changes.
func (u *User) Watch(ctx context.Context, events chan<- extensions.Event) error {
	w, err := watch.Shared()
	if err != nil {
		return err
	}
	ch, err := w.Watch("/etc/passwd", 0x00000002) // IN_MODIFY
	if err != nil {
		return err
	}
	defer w.Unwatch("/etc/passwd", ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ch:
			if !ok {
				return nil
			}
			select {
			case events <- extensions.Event{
				ResourceID: u.ID(),
				Kind:       extensions.EventWatch,
				Detail:     "inotify /etc/passwd",
				Time:       time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
