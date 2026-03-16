//go:build darwin

package plist

import (
	"context"
	"fmt"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Watch uses kqueue to monitor the plist file for changes on macOS.
func (p *Plist) Watch(ctx context.Context, events chan<- extensions.Event) error {
	path := p.plistPath()

	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("kqueue: %w", err)
	}
	defer unix.Close(kq)

	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer unix.Close(fd)

	kev := unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_VNODE,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
		Fflags: uint32(unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_ATTRIB),
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
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

		select {
		case events <- extensions.Event{
			ResourceID: p.ID(),
			Reason:     "kqueue",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
