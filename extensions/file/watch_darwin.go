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
// Re-establishes the watch after delete/rename events.
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

	fd, fflags, err := openWatch(absPath)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			if fd != -1 {
				unix.Close(fd)
				fd = -1
			}
			return nil
		default:
		}

		kev := unix.Kevent_t{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_VNODE,
			Flags:  unix.EV_ADD | unix.EV_CLEAR,
			Fflags: fflags,
		}

		timeout := unix.NsecToTimespec(int64(500 * time.Millisecond))
		out := make([]unix.Kevent_t, 1)
		n, err := unix.Kevent(kq, []unix.Kevent_t{kev}, out, &timeout)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			if fd != -1 {
				unix.Close(fd)
			}
			return fmt.Errorf("kevent: %w", err)
		}
		if n == 0 {
			continue
		}

		// Re-establish watch after delete or rename.
		if out[0].Fflags&(unix.NOTE_DELETE|unix.NOTE_RENAME) != 0 {
			unix.Close(fd)
			fd = -1
			time.Sleep(50 * time.Millisecond) // brief delay for file recreation
			newFd, newFflags, err := openWatch(absPath)
			if err == nil {
				fd = newFd
				fflags = newFflags
			}
		}

		select {
		case events <- extensions.Event{
			ResourceID: f.ID(),
			Kind: extensions.EventWatch, Detail: "kqueue",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			if fd != -1 {
				unix.Close(fd)
				fd = -1
			}
			return nil
		}
	}
}

// openWatch opens the file or its parent directory for kqueue watching.
func openWatch(absPath string) (int, uint32, error) {
	fd, err := unix.Open(absPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		dir := filepath.Dir(absPath)
		fd, err = unix.Open(dir, unix.O_RDONLY|unix.O_CLOEXEC, 0)
		if err != nil {
			return 0, 0, fmt.Errorf("open %s: %w", dir, err)
		}
		return fd, unix.NOTE_WRITE, nil
	}
	return fd, unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_RENAME | unix.NOTE_ATTRIB, nil
}
