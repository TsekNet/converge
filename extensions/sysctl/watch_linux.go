//go:build linux

package sysctl

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/watch"
	"golang.org/x/sys/unix"
)

// Watch uses the shared inotify multiplexer to monitor the sysctl value file
// under /proc/sys/.
func (s *Sysctl) Watch(ctx context.Context, events chan<- extensions.Event) error {
	if strings.Contains(s.Key, "..") {
		return fmt.Errorf("sysctl key contains path traversal: %s", s.Key)
	}

	path := filepath.Clean(procSysBase + "/" + strings.ReplaceAll(s.Key, ".", "/"))
	if !strings.HasPrefix(path, procSysBase+"/") {
		return fmt.Errorf("sysctl key escapes /proc/sys: %s", s.Key)
	}

	w, err := watch.Shared()
	if err != nil {
		return fmt.Errorf("shared inotify watcher: %w", err)
	}

	ch, err := w.Watch(path, unix.IN_MODIFY)
	if err != nil {
		return fmt.Errorf("watch %s: %w", path, err)
	}
	defer w.Unwatch(path, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ch:
		}

		select {
		case events <- extensions.Event{
			ResourceID: s.ID(),
			Kind:       extensions.EventWatch,
			Detail:     "inotify",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
