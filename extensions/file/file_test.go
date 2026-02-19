package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFile_Check(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string)
		file       func(dir string) *File
		wantSync   bool
		wantChanges int
	}{
		{
			"file does not exist",
			func(t *testing.T, dir string) {},
			func(dir string) *File {
				return New(filepath.Join(dir, "new.txt"), "hello\n", 0644)
			},
			false, 3,
		},
		{
			"file exists with correct content and mode",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("hello\n"), 0644)
			},
			func(dir string) *File {
				return New(filepath.Join(dir, "ok.txt"), "hello\n", 0644)
			},
			true, 0,
		},
		{
			"file exists with wrong content",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "wrong.txt"), []byte("old\n"), 0644)
			},
			func(dir string) *File {
				return New(filepath.Join(dir, "wrong.txt"), "new\n", 0644)
			},
			false, 1,
		},
		{
			"file exists with wrong mode",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "mode.txt"), []byte("hello\n"), 0755)
			},
			func(dir string) *File {
				return New(filepath.Join(dir, "mode.txt"), "hello\n", 0644)
			},
			false, 1,
		},
		{
			"file exists with wrong content and mode",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "both.txt"), []byte("old\n"), 0755)
			},
			func(dir string) *File {
				return New(filepath.Join(dir, "both.txt"), "new\n", 0644)
			},
			false, 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			f := tt.file(dir)

			state, err := f.Check(ctx)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			if state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", state.InSync, tt.wantSync)
			}
			if len(state.Changes) != tt.wantChanges {
				t.Errorf("len(Changes) = %d, want %d: %+v", len(state.Changes), tt.wantChanges, state.Changes)
			}
		})
	}
}

func TestFile_Apply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		file    func(dir string) *File
		verify  func(t *testing.T, dir string)
	}{
		{
			"create new file",
			func(t *testing.T, dir string) {},
			func(dir string) *File {
				return New(filepath.Join(dir, "new.txt"), "hello\n", 0644)
			},
			func(t *testing.T, dir string) {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(dir, "new.txt"))
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if string(data) != "hello\n" {
					t.Errorf("content = %q, want %q", data, "hello\n")
				}
			},
		},
		{
			"overwrite existing file",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "exist.txt"), []byte("old\n"), 0644)
			},
			func(dir string) *File {
				return New(filepath.Join(dir, "exist.txt"), "new\n", 0644)
			},
			func(t *testing.T, dir string) {
				t.Helper()
				data, _ := os.ReadFile(filepath.Join(dir, "exist.txt"))
				if string(data) != "new\n" {
					t.Errorf("content = %q, want %q", data, "new\n")
				}
			},
		},
		{
			"append to file",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "append.txt"), []byte("line1\n"), 0644)
			},
			func(dir string) *File {
				f := New(filepath.Join(dir, "append.txt"), "line2\n", 0644)
				f.Append = true
				return f
			},
			func(t *testing.T, dir string) {
				t.Helper()
				data, _ := os.ReadFile(filepath.Join(dir, "append.txt"))
				if string(data) != "line1\nline2\n" {
					t.Errorf("content = %q, want %q", data, "line1\nline2\n")
				}
			},
		},
		{
			"create nested directory",
			func(t *testing.T, dir string) {},
			func(dir string) *File {
				return New(filepath.Join(dir, "sub", "deep", "file.txt"), "nested\n", 0644)
			},
			func(t *testing.T, dir string) {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(dir, "sub", "deep", "file.txt"))
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if string(data) != "nested\n" {
					t.Errorf("content = %q", data)
				}
			},
		},
		{
			"set mode",
			func(t *testing.T, dir string) {
				t.Helper()
				os.WriteFile(filepath.Join(dir, "perm.txt"), []byte("x"), 0755)
			},
			func(dir string) *File {
				return New(filepath.Join(dir, "perm.txt"), "x", 0600)
			},
			func(t *testing.T, dir string) {
				t.Helper()
				info, _ := os.Stat(filepath.Join(dir, "perm.txt"))
				if info.Mode().Perm() != 0600 {
					t.Errorf("mode = %04o, want 0600", info.Mode().Perm())
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			f := tt.file(dir)

			result, err := f.Apply(ctx)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !result.Changed {
				t.Error("Changed should be true")
			}
			tt.verify(t, dir)
		})
	}
}

func TestFile_CheckThenApplyIdempotent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "idem.txt")

	f := New(path, "converge\n", 0644)

	state, _ := f.Check(ctx)
	if state.InSync {
		t.Fatal("should not be in sync before Apply")
	}

	f.Apply(ctx)

	f2 := New(path, "converge\n", 0644)
	state, _ = f2.Check(ctx)
	if !state.InSync {
		t.Errorf("should be in sync after Apply, changes: %+v", state.Changes)
	}
}

func TestFile_IDAndString(t *testing.T) {
	tests := []struct {
		path     string
		wantID   string
		wantStr  string
	}{
		{"/etc/motd", "file:/etc/motd", "File /etc/motd"},
		{"/tmp/test.txt", "file:/tmp/test.txt", "File /tmp/test.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			f := New(tt.path, "", 0)
			if f.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", f.ID(), tt.wantID)
			}
			if f.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", f.String(), tt.wantStr)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a ..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := truncate(tt.input, tt.max); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestFile_IsCritical(t *testing.T) {
	f := New("/tmp/test", "content", 0644)
	if f.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	f.Critical = true
	if !f.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}
