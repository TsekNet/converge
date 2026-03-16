//go:build linux

package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func waitCh(t *testing.T, ch <-chan struct{}, timeout time.Duration) {
	t.Helper()
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
	case <-time.After(timeout):
		t.Fatal("timed out waiting for notification")
	}
}

func noCh(t *testing.T, ch <-chan struct{}, wait time.Duration) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("unexpected notification")
	case <-time.After(wait):
	}
}

func TestWatchModify(t *testing.T) {
	w, err := newWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmp, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	ch, err := w.Watch(tmp, unix.IN_MODIFY)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the file.
	if err := os.WriteFile(tmp, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	waitCh(t, ch, 2*time.Second)
}

func TestWatchMultipleSubscribers(t *testing.T) {
	w, err := newWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmp, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	ch1, err := w.Watch(tmp, unix.IN_MODIFY)
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := w.Watch(tmp, unix.IN_MODIFY)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(tmp, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	waitCh(t, ch1, 2*time.Second)
	waitCh(t, ch2, 2*time.Second)
}

func TestUnwatch(t *testing.T) {
	w, err := newWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmp, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	ch, err := w.Watch(tmp, unix.IN_MODIFY)
	if err != nil {
		t.Fatal(err)
	}

	w.Unwatch(tmp, ch)

	// Channel should be closed after Unwatch.
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel after Unwatch")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after Unwatch")
	}
}

func TestReWatch(t *testing.T) {
	w, err := newWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmp, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	ch, err := w.Watch(tmp, unix.IN_MODIFY|unix.IN_DELETE_SELF)
	if err != nil {
		t.Fatal(err)
	}

	// Delete and recreate.
	os.Remove(tmp)
	// Wait for DELETE_SELF notification.
	waitCh(t, ch, 2*time.Second)

	// Recreate and re-watch.
	if err := os.WriteFile(tmp, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	w.ReWatch(tmp, unix.IN_MODIFY|unix.IN_DELETE_SELF)

	// Modify after re-watch.
	if err := os.WriteFile(tmp, []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}

	waitCh(t, ch, 2*time.Second)
}

func TestUnwatchRemovesKernelWatch(t *testing.T) {
	w, err := newWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(tmp, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	ch, err := w.Watch(tmp, unix.IN_MODIFY)
	if err != nil {
		t.Fatal(err)
	}

	w.Unwatch(tmp, ch)

	// After removing last subscriber, internal maps should be empty for this path.
	w.mu.Lock()
	_, hasSubs := w.pathSubs[tmp]
	_, hasWD := w.pathToWD[tmp]
	w.mu.Unlock()

	if hasSubs {
		t.Error("pathSubs still has entry after last Unwatch")
	}
	if hasWD {
		t.Error("pathToWD still has entry after last Unwatch")
	}

	// Modifying the file should not panic in the readLoop.
	if err := os.WriteFile(tmp, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
}
