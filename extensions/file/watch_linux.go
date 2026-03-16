//go:build linux

package file

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Watch uses inotify to monitor the file for changes. It blocks until
// ctx is cancelled, sending events when the file is modified, created,
// deleted, or has its attributes changed.
func (f *File) Watch(ctx context.Context, events chan<- extensions.Event) error {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return fmt.Errorf("inotify_init1: %w", err)
	}
	defer unix.Close(fd)

	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	// Watch the file itself if it exists.
	mask := uint32(unix.IN_MODIFY | unix.IN_CREATE | unix.IN_DELETE_SELF | unix.IN_ATTRIB | unix.IN_MOVE_SELF)
	wd, err := unix.InotifyAddWatch(fd, absPath, mask)
	if err != nil {
		// File may not exist yet: watch the parent directory instead.
		dir := filepath.Dir(absPath)
		wd, err = unix.InotifyAddWatch(fd, dir, unix.IN_CREATE|unix.IN_MOVED_TO)
		if err != nil {
			return fmt.Errorf("inotify_add_watch %s: %w", dir, err)
		}
		_ = wd
	} else {
		_ = wd
	}

	// Also watch the parent directory for file creation (handles delete + recreate).
	dir := filepath.Dir(absPath)
	unix.InotifyAddWatch(fd, dir, unix.IN_CREATE|unix.IN_MOVED_TO)

	buf := make([]byte, 4096)

	// Use epoll to make inotify reads interruptible via context cancellation.
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return fmt.Errorf("epoll_create1: %w", err)
	}
	defer unix.Close(epfd)

	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	})
	if err != nil {
		return fmt.Errorf("epoll_ctl: %w", err)
	}

	epEvents := make([]unix.EpollEvent, 1)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Wait with 500ms timeout so we can check ctx.Done() periodically.
		n, err := unix.EpollWait(epfd, epEvents, 500)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return fmt.Errorf("epoll_wait: %w", err)
		}
		if n == 0 {
			continue
		}

		// Drain inotify events.
		nBytes, err := unix.Read(fd, buf)
		if err != nil {
			if err == unix.EAGAIN {
				continue
			}
			return fmt.Errorf("read inotify: %w", err)
		}

		// Parse inotify events to generate a single notification.
		if nBytes > 0 {
			hasRelevantEvent := false
			offset := 0
			for offset < nBytes {
				event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
				offset += int(unsafe.Sizeof(*event)) + int(event.Len)
				hasRelevantEvent = true
			}

			if hasRelevantEvent {
				select {
				case events <- extensions.Event{
					ResourceID: f.ID(),
					Reason:     "inotify",
					Time:       time.Now(),
				}:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}
