//go:build darwin

package file

import (
	"context"
	"path/filepath"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Watch uses the shared kqueue multiplexer to monitor the file for changes on
// macOS. Re-establishes the watch after delete/rename events.
func (f *File) Watch(ctx context.Context, events chan<- extensions.Event) error {
	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		return err
	}

	w, err := watch.Shared()
	if err != nil {
		return err
	}

	fflags := uint32(unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_RENAME | unix.NOTE_ATTRIB)
	ch, err := w.Watch(absPath, fflags)
	if err != nil {
		return err
	}
	defer w.Unwatch(absPath, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ch:
			if !ok {
				return nil
			}
			// Re-establish watch after delete/rename: the shared watcher
			// marks the entry fd as -1, so ReWatch opens a fresh fd.
			w.ReWatch(absPath, fflags)

			select {
			case events <- extensions.Event{
				ResourceID: f.ID(),
				Kind:       extensions.EventWatch, Detail: "kqueue",
				Time: time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
