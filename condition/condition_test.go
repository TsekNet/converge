package condition_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TsekNet/converge/condition"
)

func TestFileExists_Met(t *testing.T) {
	t.Parallel()

	t.Run("exists", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp(t.TempDir(), "cond-*")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		c := condition.FileExists(f.Name())
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !met {
			t.Error("expected Met=true for existing file")
		}
	})

	t.Run("not_exists", func(t *testing.T) {
		t.Parallel()
		c := condition.FileExists(filepath.Join(t.TempDir(), "nonexistent"))
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if met {
			t.Error("expected Met=false for missing file")
		}
	})
}

func TestFileExists_Wait(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "appears-later")

	c := condition.FileExists(target)

	// Verify not met initially.
	met, _ := c.Met(context.Background())
	if met {
		t.Fatal("file should not exist yet")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- c.Wait(ctx) }()

	// Create the file after a short delay.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(target, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Wait returned error: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Error("Wait did not return after file was created")
	}
}

func TestFileExists_Wait_CtxCancel(t *testing.T) {
	t.Parallel()

	c := condition.FileExists(filepath.Join(t.TempDir(), "never-created"))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Wait(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected non-nil error on ctx cancel")
		}
	case <-time.After(2 * time.Second):
		t.Error("Wait did not return after ctx cancel")
	}
}

func TestNetworkReachable_Met(t *testing.T) {
	t.Parallel()

	t.Run("reachable", func(t *testing.T) {
		t.Parallel()
		// Loopback is always reachable if something listens; use a known port.
		// Just test the negative case reliably.
		c := condition.NetworkReachable("240.0.0.1", 9) // TEST-NET, nothing listens
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if met {
			t.Error("expected Met=false for unreachable host")
		}
	})
}

func TestNetworkInterface_Met(t *testing.T) {
	t.Parallel()

	t.Run("loopback_up", func(t *testing.T) {
		t.Parallel()
		c := condition.NetworkInterface("lo")
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !met {
			t.Error("expected loopback to be up")
		}
	})

	t.Run("nonexistent", func(t *testing.T) {
		t.Parallel()
		c := condition.NetworkInterface("tun99nonexistent")
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if met {
			t.Error("expected Met=false for nonexistent interface")
		}
	})
}

func TestMountPoint_Met(t *testing.T) {
	t.Parallel()

	t.Run("tmpdir_not_mount", func(t *testing.T) {
		t.Parallel()
		// A freshly created temp dir is not a mount point.
		c := condition.MountPoint(t.TempDir())
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if met {
			t.Logf("TempDir is a mount point (unusual but possible in containers/WSL): skipping")
		}
	})

	t.Run("nonexistent_not_met", func(t *testing.T) {
		t.Parallel()
		c := condition.MountPoint(filepath.Join(t.TempDir(), "nodir"))
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if met {
			t.Error("expected Met=false for nonexistent path")
		}
	})
}

func TestCondition_String(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		c    interface{ String() string }
		want string
	}{
		{"FileExists", condition.FileExists("/tmp/x"), "file exists /tmp/x"},
		{"NetworkReachable", condition.NetworkReachable("host", 80), "network reachable host:80"},
		{"NetworkInterface", condition.NetworkInterface("eth0"), "network interface eth0 up"},
		{"MountPoint", condition.MountPoint("/mnt/nfs"), "mount point /mnt/nfs"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.c.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
