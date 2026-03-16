//go:build darwin

package file

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Watch uses kqueue to monitor the file for changes on macOS.
func (f *File) Watch(ctx context.Context, events chan<- extensions.Event) error {
	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("kqueue: %w", err)
	}
	defer unix.Close(kq)

	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	// Try to open the file itself, fall back to parent dir if it doesn't exist.
	fd, err := unix.Open(absPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	watchingDir := false
	if err != nil {
		dir := filepath.Dir(absPath)
		fd, err = unix.Open(dir, unix.O_RDONLY|unix.O_CLOEXEC, 0)
		if err != nil {
			return fmt.Errorf("open %s: %w", dir, err)
		}
		watchingDir = true
	}
	defer unix.Close(fd)

	fflags := uint32(unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_RENAME | unix.NOTE_ATTRIB)
	if watchingDir {
		fflags = unix.NOTE_WRITE // directory write = file created/deleted inside
	}

	kev := unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_VNODE,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
		Fflags: fflags,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Use a timeout so we can check ctx.Done() periodically.
		timeout := unix.NsecToTimespec(int64(500 * time.Millisecond))
		out := make([]unix.Kevent_t, 1)
		n, err := unix.Kevent(kq, []unix.Kevent_t{kev}, out, &timeout)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return fmt.Errorf("kevent: %w", err)
		}
		if n == 0 {
			continue
		}

		select {
		case events <- extensions.Event{
			ResourceID: f.ID(),
			Reason:     "kqueue",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
