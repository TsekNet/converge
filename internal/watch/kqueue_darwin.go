//go:build darwin

// Package watch provides a shared kqueue multiplexer that uses a single kqueue
// fd for all file/plist watchers, avoiding per-resource fd exhaustion.
package watch

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// kqSubscriber holds the notification channel and fflags for one caller.
type kqSubscriber struct {
	ch     chan struct{}
	fflags uint32
}

// watchEntry tracks an open fd and its subscribers within the kqueue watcher.
type watchEntry struct {
	fd   int
	subs []*kqSubscriber
}

// KqueueWatcher multiplexes many watch paths onto a single kqueue fd.
// Safe for concurrent use.
type KqueueWatcher struct {
	mu sync.Mutex
	kq int // kqueue fd

	// fd -> path, for mapping kqueue events back.
	fdToPath map[int]string
	// path -> watchEntry (open fd + subscribers).
	pathEntry map[string]*watchEntry

	running bool
	done    chan struct{}
}

var (
	globalWatcher *KqueueWatcher
	globalOnce    sync.Once
	globalErr     error
)

// Shared returns the process-wide shared KqueueWatcher, creating it on first
// call. Returns an error only if the underlying kqueue syscall fails.
func Shared() (*KqueueWatcher, error) {
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

func newWatcher() (*KqueueWatcher, error) {
	kq, err := unix.Kqueue()
	if err != nil {
		return nil, fmt.Errorf("kqueue: %w", err)
	}

	return &KqueueWatcher{
		kq:        kq,
		fdToPath:  make(map[int]string),
		pathEntry: make(map[string]*watchEntry),
		done:      make(chan struct{}),
	}, nil
}

// Watch adds a kqueue watch for path with the given fflags and returns a
// channel that receives a struct{}{} each time a matching event fires.
// If the file does not exist, the parent directory is watched instead.
// Multiple calls with the same path share a single kernel kevent (OR of all
// fflags). The returned channel is buffered (capacity 1) so a slow consumer
// does not block other subscribers.
func (w *KqueueWatcher) Watch(path string, fflags uint32) (<-chan struct{}, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan struct{}, 1)
	sub := &kqSubscriber{ch: ch, fflags: fflags}

	entry, exists := w.pathEntry[absPath]
	if exists {
		entry.subs = append(entry.subs, sub)
		// Re-register kevent with merged fflags.
		w.registerKevent(entry, absPath)
		return ch, nil
	}

	// Open the file, or fall back to parent directory.
	fd, actualFflags, err := openForKqueue(absPath, fflags)
	if err != nil {
		return nil, err
	}

	sub.fflags = actualFflags
	entry = &watchEntry{
		fd:   fd,
		subs: []*kqSubscriber{sub},
	}
	w.pathEntry[absPath] = entry
	w.fdToPath[fd] = absPath

	w.registerKevent(entry, absPath)

	if !w.running {
		w.running = true
		go w.readLoop()
	}

	return ch, nil
}

// registerKevent registers or updates the kevent for an entry.
func (w *KqueueWatcher) registerKevent(entry *watchEntry, path string) {
	merged := uint32(0)
	for _, s := range entry.subs {
		merged |= s.fflags
	}

	kev := unix.Kevent_t{
		Ident:  uint64(entry.fd),
		Filter: unix.EVFILT_VNODE,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
		Fflags: merged,
	}
	// Ignore error: best effort.
	unix.Kevent(w.kq, []unix.Kevent_t{kev}, nil, nil)
}

// Unwatch removes one subscriber channel for path. When the last subscriber
// for a path is removed, the file descriptor is closed and the kevent removed.
func (w *KqueueWatcher) Unwatch(path string, ch <-chan struct{}) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	entry, ok := w.pathEntry[absPath]
	if !ok {
		return
	}

	for i, s := range entry.subs {
		if s.ch == ch {
			entry.subs[i] = entry.subs[len(entry.subs)-1]
			entry.subs = entry.subs[:len(entry.subs)-1]
			close(s.ch)
			break
		}
	}

	if len(entry.subs) == 0 {
		// Remove kevent by deleting the fd.
		kev := unix.Kevent_t{
			Ident:  uint64(entry.fd),
			Filter: unix.EVFILT_VNODE,
			Flags:  unix.EV_DELETE,
		}
		unix.Kevent(w.kq, []unix.Kevent_t{kev}, nil, nil)
		unix.Close(entry.fd)
		delete(w.fdToPath, entry.fd)
		delete(w.pathEntry, absPath)
	} else {
		// Re-register with updated merged fflags.
		w.registerKevent(entry, absPath)
	}
}

// ReWatch re-establishes a watch for path after it was deleted/moved. This is
// a no-op if no subscribers remain for the path.
func (w *KqueueWatcher) ReWatch(path string, fflags uint32) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	entry, ok := w.pathEntry[absPath]
	if !ok || len(entry.subs) == 0 {
		return
	}

	// Close old fd if still open.
	if entry.fd != -1 {
		kev := unix.Kevent_t{
			Ident:  uint64(entry.fd),
			Filter: unix.EVFILT_VNODE,
			Flags:  unix.EV_DELETE,
		}
		unix.Kevent(w.kq, []unix.Kevent_t{kev}, nil, nil)
		unix.Close(entry.fd)
		delete(w.fdToPath, entry.fd)
	}

	fd, actualFflags, err := openForKqueue(absPath, fflags)
	if err != nil {
		// File hasn't reappeared yet; mark fd invalid.
		entry.fd = -1
		return
	}

	entry.fd = fd
	// Update subscriber fflags to match what was actually opened.
	for _, s := range entry.subs {
		s.fflags = actualFflags
	}
	w.fdToPath[fd] = absPath
	w.registerKevent(entry, absPath)
}

// Close shuts down the watcher, closing the kqueue fd.
// All subscriber channels are closed.
func (w *KqueueWatcher) Close() error {
	w.mu.Lock()
	for _, entry := range w.pathEntry {
		for _, s := range entry.subs {
			close(s.ch)
		}
		if entry.fd != -1 {
			unix.Close(entry.fd)
		}
	}
	w.pathEntry = make(map[string]*watchEntry)
	w.fdToPath = make(map[int]string)
	w.mu.Unlock()

	return unix.Close(w.kq)
}

func (w *KqueueWatcher) readLoop() {
	out := make([]unix.Kevent_t, 32)
	timeout := unix.NsecToTimespec(int64(500 * time.Millisecond))

	for {
		n, err := unix.Kevent(w.kq, nil, out, &timeout)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return // kqueue fd closed or fatal error
		}
		if n == 0 {
			continue
		}

		w.mu.Lock()
		for i := 0; i < n; i++ {
			ev := out[i]
			fd := int(ev.Ident)

			path, ok := w.fdToPath[fd]
			if !ok {
				continue
			}

			entry := w.pathEntry[path]
			if entry == nil {
				continue
			}

			// Notify subscribers.
			for _, s := range entry.subs {
				select {
				case s.ch <- struct{}{}:
				default:
				}
			}

			// If the file was deleted or renamed, mark the entry for rewatch.
			if ev.Fflags&(unix.NOTE_DELETE|unix.NOTE_RENAME) != 0 {
				unix.Close(entry.fd)
				delete(w.fdToPath, entry.fd)
				entry.fd = -1
			}
		}
		w.mu.Unlock()
	}
}

// openForKqueue opens the file for kqueue watching. If the file does not
// exist, falls back to the parent directory with NOTE_WRITE only.
func openForKqueue(absPath string, fflags uint32) (int, uint32, error) {
	fd, err := unix.Open(absPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		dir := filepath.Dir(absPath)
		fd, err = unix.Open(dir, unix.O_RDONLY|unix.O_CLOEXEC, 0)
		if err != nil {
			return 0, 0, fmt.Errorf("open %s: %w", dir, err)
		}
		return fd, unix.NOTE_WRITE, nil
	}
	return fd, fflags, nil
}
