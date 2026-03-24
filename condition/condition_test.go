package condition_test

import (
	"context"
	"net"
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

	t.Run("unreachable", func(t *testing.T) {
		t.Parallel()
		c := condition.NetworkReachable("240.0.0.1", 9) // TEST-NET, nothing listens
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if met {
			t.Error("expected Met=false for unreachable host")
		}
	})

	t.Run("reachable", func(t *testing.T) {
		t.Parallel()
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		addr := ln.Addr().(*net.TCPAddr)
		c := condition.NetworkReachable("127.0.0.1", addr.Port)
		met, err := c.Met(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !met {
			t.Error("expected Met=true for listening port")
		}
	})
}

func TestNetworkInterface_Met(t *testing.T) {
	t.Parallel()

	t.Run("loopback_up", func(t *testing.T) {
		t.Parallel()
		ifaces, err := net.Interfaces()
		if err != nil {
			t.Fatal(err)
		}
		var loopback string
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				loopback = iface.Name
				break
			}
		}
		if loopback == "" {
			t.Skip("no loopback interface found")
		}
		c := condition.NetworkInterface(loopback)
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
			t.Skip("TempDir is a mount point (unusual but possible in containers/WSL)")
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

func TestNetworkReachable_Wait_AlreadyMet(t *testing.T) {
	t.Parallel()

	// Start a listener before creating the condition, so it is already met.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)

	c := condition.NetworkReachable("127.0.0.1", addr.Port)

	// Confirm Met is true before calling Wait.
	met, err := c.Met(context.Background())
	if err != nil {
		t.Fatalf("Met() error = %v", err)
	}
	if !met {
		t.Fatal("expected condition to be already met")
	}

	// Wait should return immediately since the condition is already satisfied.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	if err := c.Wait(ctx); err != nil {
		t.Errorf("Wait() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 1*time.Second {
		t.Errorf("Wait() took %v, expected immediate return", elapsed)
	}
}

func TestNetworkReachable_Wait_CtxCancel(t *testing.T) {
	t.Parallel()

	c := condition.NetworkReachable("240.0.0.1", 9)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Wait(ctx) }()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected non-nil error on ctx cancel")
		}
	case <-time.After(5 * time.Second):
		t.Error("Wait did not return after ctx cancel")
	}
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
