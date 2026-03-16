//go:build darwin

package plist

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Watch uses kqueue to monitor the plist file for changes on macOS.
// Falls back to watching the parent directory if the file doesn't exist yet.
func (p *Plist) Watch(ctx context.Context, events chan<- extensions.Event) error {
	path := p.plistPath()

	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("kqueue: %w", err)
	}
	defer unix.Close(kq)

	fd, fflags, err := openPlistWatch(path)
	if err != nil {
		return err
	}
	defer func() { unix.Close(fd) }()

	for {
		select {
		case <-ctx.Done():
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
			return fmt.Errorf("kevent: %w", err)
		}
		if n == 0 {
			continue
		}

		// Re-establish watch after delete.
		if out[0].Fflags&(unix.NOTE_DELETE|unix.NOTE_RENAME) != 0 {
			unix.Close(fd)
			time.Sleep(50 * time.Millisecond)
			newFd, newFflags, err := openPlistWatch(path)
			if err == nil {
				fd = newFd
				fflags = newFflags
			}
		}

		select {
		case events <- extensions.Event{
			ResourceID: p.ID(),
			Kind: extensions.EventWatch, Detail: "kqueue",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}

// openPlistWatch opens the plist file or its parent directory for kqueue.
func openPlistWatch(path string) (int, uint32, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		dir := filepath.Dir(path)
		fd, err = unix.Open(dir, unix.O_RDONLY|unix.O_CLOEXEC, 0)
		if err != nil {
			return 0, 0, fmt.Errorf("open %s: %w", dir, err)
		}
		return fd, unix.NOTE_WRITE, nil
	}
	return fd, unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_ATTRIB | unix.NOTE_RENAME, nil
}
