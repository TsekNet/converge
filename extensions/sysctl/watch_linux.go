//go:build linux

package sysctl

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Watch uses inotify to monitor the sysctl value file under /proc/sys/.
func (s *Sysctl) Watch(ctx context.Context, events chan<- extensions.Event) error {
	path := procSysBase + "/" + strings.ReplaceAll(s.Key, ".", "/")

	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return fmt.Errorf("inotify_init1: %w", err)
	}
	defer unix.Close(fd)

	_, err = unix.InotifyAddWatch(fd, path, unix.IN_MODIFY)
	if err != nil {
		return fmt.Errorf("inotify_add_watch %s: %w", path, err)
	}

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return fmt.Errorf("epoll_create1: %w", err)
	}
	defer unix.Close(epfd)

	unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	})

	buf := make([]byte, 4096)
	epEvents := make([]unix.EpollEvent, 1)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

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

		nBytes, err := unix.Read(fd, buf)
		if err == unix.EAGAIN || nBytes == 0 {
			continue
		}
		if err != nil {
			return fmt.Errorf("read inotify: %w", err)
		}

		// Drain all events.
		offset := 0
		for offset < nBytes {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			offset += int(unsafe.Sizeof(*event)) + int(event.Len)
		}

		select {
		case events <- extensions.Event{
			ResourceID: s.ID(),
			Reason:     "inotify",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
