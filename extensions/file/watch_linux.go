//go:build linux

package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Watch uses the shared inotify multiplexer to monitor the file for changes.
// It blocks until ctx is cancelled, sending events when the file is modified,
// created, deleted, or has its attributes changed. Re-establishes the watch
// after delete/move events.
func (f *File) Watch(ctx context.Context, events chan<- extensions.Event) error {
	w, err := watch.Shared()
	if err != nil {
		return fmt.Errorf("shared inotify watcher: %w", err)
	}

	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}
	dir := filepath.Dir(absPath)

	// Watch the parent directory for file creation/move-in.
	dirCh, err := w.Watch(dir, unix.IN_CREATE|unix.IN_MOVED_TO)
	if err != nil {
		return fmt.Errorf("watch dir %s: %w", dir, err)
	}
	defer w.Unwatch(dir, dirCh)

	// Watch the file itself if it exists.
	fileMask := uint32(unix.IN_MODIFY | unix.IN_CREATE | unix.IN_DELETE_SELF | unix.IN_ATTRIB | unix.IN_MOVE_SELF)
	var fileCh <-chan struct{}
	if _, err := os.Stat(absPath); err == nil {
		fileCh, err = w.Watch(absPath, fileMask)
		if err != nil {
			return fmt.Errorf("watch file %s: %w", absPath, err)
		}
		defer w.Unwatch(absPath, fileCh)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-dirCh:
			// A file appeared in the directory: try to re-establish file watch.
			if fileCh == nil {
				if _, statErr := os.Stat(absPath); statErr == nil {
					newCh, watchErr := w.Watch(absPath, fileMask)
					if watchErr == nil {
						fileCh = newCh
						defer w.Unwatch(absPath, fileCh)
					}
				}
			}
		case <-fileCh:
			// Re-establish watch if file was deleted/moved.
			w.ReWatch(absPath, fileMask)
		}

		select {
		case events <- extensions.Event{
			ResourceID: f.ID(),
			Kind:       extensions.EventWatch,
			Detail:     "inotify",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
