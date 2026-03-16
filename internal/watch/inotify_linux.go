//go:build linux

// Package watch provides a shared inotify multiplexer that uses a single
// inotify fd and epoll fd for all file watchers, avoiding per-resource fd
// exhaustion against inotify_max_user_instances (default 128).
package watch

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

// subscriber holds the notification channel and inotify mask for one caller.
type subscriber struct {
	ch   chan struct{}
	mask uint32
}

// InotifyWatcher multiplexes many watch paths onto a single inotify fd.
// Safe for concurrent use.
type InotifyWatcher struct {
	mu   sync.Mutex
	fd   int // inotify fd
	epfd int // epoll fd

	// wd -> path, for mapping inotify events back.
	wdToPath map[int32]string
	// path -> wd, for removal.
	pathToWD map[string]int32
	// path -> list of subscribers.
	pathSubs map[string][]*subscriber

	running bool
	done    chan struct{}
}

var (
	globalWatcher *InotifyWatcher
	globalOnce    sync.Once
	globalErr     error
)

// Shared returns the process-wide shared InotifyWatcher, creating it on first
// call. Returns an error only if the underlying inotify/epoll syscalls fail.
func Shared() (*InotifyWatcher, error) {
	globalOnce.Do(func() {
		w, err := newWatcher()
		if err != nil {
			globalErr = err
			return
		}
		globalWatcher = w
	})
	return globalWatcher, globalErr
}

func newWatcher() (*InotifyWatcher, error) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, fmt.Errorf("inotify_init1: %w", err)
	}

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("epoll_create1: %w", err)
	}

	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	}); err != nil {
		unix.Close(epfd)
		unix.Close(fd)
		return nil, fmt.Errorf("epoll_ctl: %w", err)
	}

	return &InotifyWatcher{
		fd:       fd,
		epfd:     epfd,
		wdToPath: make(map[int32]string),
		pathToWD: make(map[string]int32),
		pathSubs: make(map[string][]*subscriber),
		done:     make(chan struct{}),
	}, nil
}

// Watch adds an inotify watch for path with the given mask and returns a
// channel that receives a struct{}{} each time a matching event fires.
// Multiple calls with the same path merge into a single kernel watch (OR of
// all masks). The returned channel is buffered (capacity 1) so a slow consumer
// does not block other subscribers.
func (w *InotifyWatcher) Watch(path string, mask uint32) (<-chan struct{}, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan struct{}, 1)
	sub := &subscriber{ch: ch, mask: mask}

	// Merge masks if path is already watched.
	merged := mask
	for _, s := range w.pathSubs[path] {
		merged |= s.mask
	}

	wd, err := unix.InotifyAddWatch(w.fd, path, merged)
	if err != nil {
		return nil, fmt.Errorf("inotify_add_watch %s: %w", path, err)
	}

	// Clean up old wd mapping if the kernel reassigned it.
	if oldPath, ok := w.wdToPath[int32(wd)]; ok && oldPath != path {
		delete(w.pathToWD, oldPath)
	}

	w.wdToPath[int32(wd)] = path
	w.pathToWD[path] = int32(wd)
	w.pathSubs[path] = append(w.pathSubs[path], sub)

	if !w.running {
		w.running = true
		go w.readLoop()
	}

	return ch, nil
}

// Unwatch removes one subscriber channel for path. When the last subscriber
// for a path is removed, the kernel watch is also removed.
func (w *InotifyWatcher) Unwatch(path string, ch <-chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	subs := w.pathSubs[path]
	for i, s := range subs {
		if s.ch == ch {
			subs[i] = subs[len(subs)-1]
			subs = subs[:len(subs)-1]
			close(s.ch)
			break
		}
	}

	if len(subs) == 0 {
		delete(w.pathSubs, path)
		if wd, ok := w.pathToWD[path]; ok {
			// Ignore error: wd may already be auto-removed (IN_DELETE_SELF).
			unix.InotifyRmWatch(w.fd, uint32(wd))
			delete(w.wdToPath, wd)
			delete(w.pathToWD, path)
		}
	} else {
		w.pathSubs[path] = subs
	}
}

// ReWatch re-establishes a watch for path after it was deleted/moved. This is
// a no-op if no subscribers remain for the path.
func (w *InotifyWatcher) ReWatch(path string, mask uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.pathSubs[path]) == 0 {
		return
	}

	// Merge all subscriber masks.
	merged := mask
	for _, s := range w.pathSubs[path] {
		merged |= s.mask
	}

	wd, err := unix.InotifyAddWatch(w.fd, path, merged)
	if err != nil {
		return // file hasn't reappeared yet
	}

	if oldPath, ok := w.wdToPath[int32(wd)]; ok && oldPath != path {
		delete(w.pathToWD, oldPath)
	}
	w.wdToPath[int32(wd)] = path
	w.pathToWD[path] = int32(wd)
}

// Close shuts down the watcher, closing both the inotify and epoll fds.
// All subscriber channels are closed.
func (w *InotifyWatcher) Close() error {
	w.mu.Lock()
	for _, subs := range w.pathSubs {
		for _, s := range subs {
			close(s.ch)
		}
	}
	w.pathSubs = make(map[string][]*subscriber)
	w.wdToPath = make(map[int32]string)
	w.pathToWD = make(map[string]int32)
	w.mu.Unlock()

	// Closing the fds causes the readLoop to exit.
	err1 := unix.Close(w.epfd)
	err2 := unix.Close(w.fd)
	if err1 != nil {
		return err1
	}
	return err2
}

func (w *InotifyWatcher) readLoop() {
	buf := make([]byte, 4096)
	epEvents := make([]unix.EpollEvent, 1)
	eventSize := int(unsafe.Sizeof(unix.InotifyEvent{}))

	for {
		n, err := unix.EpollWait(w.epfd, epEvents, 500)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return // fd closed or fatal error
		}
		if n == 0 {
			continue
		}

		nBytes, err := unix.Read(w.fd, buf)
		if err == unix.EAGAIN || nBytes == 0 {
			continue
		}
		if err != nil {
			return // fd closed
		}

		// Bounds check: need at least one full event header.
		if nBytes < eventSize {
			continue
		}

		w.mu.Lock()
		offset := 0
		for offset+eventSize <= nBytes {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))

			// Overflow guard: ensure the full event (header + name) fits.
			if offset+eventSize+int(event.Len) > nBytes {
				break
			}

			offset += eventSize + int(event.Len)

			path, ok := w.wdToPath[event.Wd]
			if !ok {
				continue
			}

			for _, s := range w.pathSubs[path] {
				// Non-blocking send: drop if channel already has a pending notification.
				select {
				case s.ch <- struct{}{}:
				default:
				}
			}
		}
		w.mu.Unlock()
	}
}
