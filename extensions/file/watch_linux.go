//go:build linux

package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Watch uses inotify to monitor the file for changes. It blocks until
// ctx is cancelled, sending events when the file is modified, created,
// deleted, or has its attributes changed. Re-establishes the watch
// after delete/move events.
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
	dir := filepath.Dir(absPath)

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return fmt.Errorf("epoll_create1: %w", err)
	}
	defer unix.Close(epfd)

	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	}); err != nil {
		return fmt.Errorf("epoll_ctl: %w", err)
	}

	// Always watch the parent directory for file creation.
	if _, err := unix.InotifyAddWatch(fd, dir, unix.IN_CREATE|unix.IN_MOVED_TO); err != nil {
		return fmt.Errorf("inotify_add_watch dir %s: %w", dir, err)
	}

	// Watch the file itself if it exists.
	fileMask := uint32(unix.IN_MODIFY | unix.IN_CREATE | unix.IN_DELETE_SELF | unix.IN_ATTRIB | unix.IN_MOVE_SELF)
	addFileWatch := func() {
		unix.InotifyAddWatch(fd, absPath, fileMask)
	}
	if _, err := os.Stat(absPath); err == nil {
		addFileWatch()
	}

	buf := make([]byte, 4096)
	epEvents := make([]unix.EpollEvent, 1)
	eventSize := int(unsafe.Sizeof(unix.InotifyEvent{}))

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

		needsRewire := false
		offset := 0
		for offset+eventSize <= nBytes {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			offset += eventSize + int(event.Len)

			if event.Mask&(unix.IN_DELETE_SELF|unix.IN_MOVE_SELF) != 0 {
				needsRewire = true
			}
		}

		// Re-establish watch after delete/move.
		if needsRewire {
			addFileWatch()
		}

		select {
		case events <- extensions.Event{
			ResourceID: f.ID(),
			Kind: extensions.EventWatch, Detail: "inotify",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
