//go:build darwin

package plist

import (
	"context"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Watch uses the shared kqueue multiplexer to monitor the plist file for
// changes on macOS. Falls back to watching the parent directory if the file
// doesn't exist yet.
func (p *Plist) Watch(ctx context.Context, events chan<- extensions.Event) error {
	path := p.plistPath()

	w, err := watch.Shared()
	if err != nil {
		return err
	}

	fflags := uint32(unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_ATTRIB | unix.NOTE_RENAME)
	ch, err := w.Watch(path, fflags)
	if err != nil {
		return err
	}
	defer w.Unwatch(path, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ch:
			if !ok {
				return nil
			}
			// Re-establish watch after delete/rename.
			w.ReWatch(path, fflags)

			select {
			case events <- extensions.Event{
				ResourceID: p.ID(),
				Kind:       extensions.EventWatch, Detail: "kqueue",
				Time: time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
