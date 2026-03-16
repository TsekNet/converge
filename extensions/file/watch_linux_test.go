//go:build linux

package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TsekNet/converge/extensions"
)

func TestWatch_DetectsModification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("initial"), 0644)

	f := New(path, "desired", 0644)
	events := make(chan extensions.Event, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go f.Watch(ctx, events)

	// Give inotify time to set up.
	time.Sleep(100 * time.Millisecond)

	// Modify the file externally.
	os.WriteFile(path, []byte("modified"), 0644)

	select {
	case evt := <-events:
		if evt.ResourceID != f.ID() {
			t.Errorf("event resource = %q, want %q", evt.ResourceID, f.ID())
		}
		if evt.Reason != "inotify" {
			t.Errorf("event reason = %q, want inotify", evt.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for inotify event")
	}
}

func TestWatch_DetectsCreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "newfile.txt")

	f := New(path, "content", 0644)
	events := make(chan extensions.Event, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go f.Watch(ctx, events)

	time.Sleep(100 * time.Millisecond)

	// Create the file externally.
	os.WriteFile(path, []byte("created"), 0644)

	select {
	case evt := <-events:
		if evt.ResourceID != f.ID() {
			t.Errorf("event resource = %q, want %q", evt.ResourceID, f.ID())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for inotify event on file creation")
	}
}

func TestWatch_CancelsCleanly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("data"), 0644)

	f := New(path, "data", 0644)
	events := make(chan extensions.Event, 10)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- f.Watch(ctx, events)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Watch returned error on cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not return after context cancellation")
	}
}
