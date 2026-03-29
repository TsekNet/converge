package reboot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestNew_sanitizesName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"../../etc/passwd", "passwd"},
		{"a/b/c", "c"},
		{`a\b\c`, "c"},
		{`a/b\c`, "c"},
		{`a\b/c`, "c"},
		{"foo/", "foo"}, // filepath.Base strips trailing slash
		{"///", "/"},    // filepath.Base("///") = "/"
		{".", ""},       // filepath.Base(".") = "." which is rejected
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			r := New(tt.input)
			if r.Name != tt.want {
				t.Errorf("New(%q).Name = %q, want %q", tt.input, r.Name, tt.want)
			}
		})
	}
}

func TestReboot_ID(t *testing.T) {
	t.Parallel()
	r := New("kernel-update")
	if got := r.ID(); got != "reboot:kernel-update" {
		t.Errorf("ID() = %q, want %q", got, "reboot:kernel-update")
	}
}

func TestReboot_String(t *testing.T) {
	t.Parallel()
	r := New("kernel-update")
	if got := r.String(); got != "Reboot kernel-update" {
		t.Errorf("String() = %q, want %q", got, "Reboot kernel-update")
	}
}

func TestReboot_IsCritical(t *testing.T) {
	t.Parallel()
	r := New("test")
	if r.IsCritical() {
		t.Error("default IsCritical() should be false")
	}
	r.Critical = true
	if !r.IsCritical() {
		t.Error("IsCritical() should be true after setting Critical")
	}
}

func TestEffectiveMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		reason  string
		message string
		want    string
	}{
		{"message set", "the reason", "the message", "the message"},
		{"message empty", "the reason", "", "the reason"},
		{"both empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Reboot{Reason: tt.reason, Message: tt.message}
			if got := r.effectiveMessage(); got != tt.want {
				t.Errorf("effectiveMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteSentinel_createsDirs(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sub", "deep", "test.sentinel")
	if err := writeSentinel(path); err != nil {
		t.Fatalf("writeSentinel() error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("sentinel file not created: %v", err)
	}
}

func TestSentinelTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name:    "second precision (current format)",
			content: strconv.FormatInt(time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC).Unix(), 10),
			check: func(t *testing.T, got time.Time) {
				t.Helper()
				want := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:    "nanosecond precision (legacy format)",
			content: "1735689600000000000", // 2025-01-01T00:00:00Z
			check: func(t *testing.T, got time.Time) {
				t.Helper()
				want := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:    "invalid content",
			content: "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "test.sentinel")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := sentinelTime(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("sentinelTime() error: %v", err)
			}
			tt.check(t, got)
		})
	}
}

func TestSentinelTime_missing(t *testing.T) {
	t.Parallel()
	got, err := sentinelTime(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("sentinelTime() error: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("expected zero time for missing sentinel, got %v", got)
	}
}

func TestWriteSentinel_roundtrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.sentinel")
	before := time.Now()
	if err := writeSentinel(path); err != nil {
		t.Fatalf("writeSentinel() error: %v", err)
	}
	got, err := sentinelTime(path)
	if err != nil {
		t.Fatalf("sentinelTime() error: %v", err)
	}
	if got.Before(before.Truncate(time.Second)) {
		t.Errorf("sentinel time %v is before write time %v", got, before)
	}
	if got.After(time.Now().Add(time.Second)) {
		t.Errorf("sentinel time %v is in the future", got)
	}
}

func TestCheckState(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name        string
		sentinel    string // "" = no sentinel, otherwise written to the sentinel file
		bootTime    time.Time
		wantSync    bool
		wantChanges int
	}{
		{
			name:        "no sentinel: drifted",
			sentinel:    "",
			bootTime:    now,
			wantSync:    false,
			wantChanges: 1,
		},
		{
			name:        "boot well after sentinel: compliant",
			sentinel:    fmt.Sprintf("%d", now.Add(-1*time.Hour).Unix()),
			bootTime:    now,
			wantSync:    true,
			wantChanges: 0,
		},
		{
			name:        "boot within 2s grace of sentinel: compliant",
			sentinel:    fmt.Sprintf("%d", now.Unix()),
			bootTime:    now.Add(-1 * time.Second),
			wantSync:    true,
			wantChanges: 0,
		},
		{
			name:        "boot before sentinel: drifted (reboot pending)",
			sentinel:    fmt.Sprintf("%d", now.Unix()),
			bootTime:    now.Add(-1 * time.Hour),
			wantSync:    false,
			wantChanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "reboot-test.sentinel")

			r := New("test")
			r.sentinelOverride = path

			if tt.sentinel != "" {
				if err := os.WriteFile(path, []byte(tt.sentinel), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			state, err := r.checkState(tt.bootTime)
			if err != nil {
				t.Fatalf("checkState() error: %v", err)
			}
			if state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", state.InSync, tt.wantSync)
			}
			if len(state.Changes) != tt.wantChanges {
				t.Errorf("len(Changes) = %d, want %d", len(state.Changes), tt.wantChanges)
			}
		})
	}
}

func TestRemoveSentinel(t *testing.T) {
	t.Parallel()

	t.Run("removes existing sentinel", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "reboot-test.sentinel")
		if err := os.WriteFile(path, []byte("123"), 0o644); err != nil {
			t.Fatal(err)
		}

		r := New("test")
		r.sentinelOverride = path
		if err := r.removeSentinel(); err != nil {
			t.Fatalf("removeSentinel() error: %v", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("sentinel file still exists after removal")
		}
	})

	t.Run("no error for missing sentinel", func(t *testing.T) {
		t.Parallel()
		r := New("test")
		r.sentinelOverride = filepath.Join(t.TempDir(), "nonexistent.sentinel")
		if err := r.removeSentinel(); err != nil {
			t.Errorf("removeSentinel() on missing file: %v", err)
		}
	})
}
