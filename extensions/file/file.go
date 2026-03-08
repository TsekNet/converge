package file

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

// File manages content, permissions, and ownership of a file on disk.
type File struct {
	Path    string
	Content string
	Mode    fs.FileMode
	Owner   string
	Group   string
	Append  bool
	Critical bool
}

func New(path string, content string, mode fs.FileMode) *File {
	return &File{Path: path, Content: content, Mode: mode}
}

func (f *File) ID() string       { return fmt.Sprintf("file:%s", f.Path) }
func (f *File) String() string   { return fmt.Sprintf("File %s", f.Path) }
func (f *File) IsCritical() bool { return f.Critical }

func (f *File) Check(_ context.Context) (*extensions.State, error) {
	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", f.Path, err)
	}
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		changes := []extensions.Change{
			{Property: "state", To: "create", Action: "add"},
			{Property: "content", To: summarizeContent(f.Content), Action: "add"},
		}
		if f.Mode != 0 {
			changes = append(changes, extensions.Change{
				Property: "mode", To: fmt.Sprintf("%04o", f.Mode), Action: "add",
			})
		}
		return &extensions.State{InSync: false, Changes: changes}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", absPath, err)
	}

	var changes []extensions.Change

	if f.Content != "" && !f.Append {
		existing, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", absPath, err)
		}
		if string(existing) != f.Content {
			changes = append(changes, diffContent(string(existing), f.Content)...)
		}
	}

	if f.Mode != 0 && info.Mode().Perm() != f.Mode {
		changes = append(changes, extensions.Change{
			Property: "mode",
			From:     fmt.Sprintf("%04o", info.Mode().Perm()),
			To:       fmt.Sprintf("%04o", f.Mode),
			Action:   "modify",
		})
	}

	return &extensions.State{InSync: len(changes) == 0, Changes: changes}, nil
}

func (f *File) Apply(_ context.Context) (*extensions.Result, error) {
	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}

	if f.Content != "" {
		flag := os.O_WRONLY | os.O_CREATE
		if f.Append {
			flag |= os.O_APPEND
		} else {
			flag |= os.O_TRUNC
		}
		file, err := os.OpenFile(f.Path, flag, 0644)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f.Path, err)
		}
		_, writeErr := file.WriteString(f.Content)
		closeErr := file.Close()
		if writeErr != nil {
			return nil, fmt.Errorf("write %s: %w", f.Path, writeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s: %w", f.Path, closeErr)
		}
	}

	if f.Mode != 0 {
		if err := os.Chmod(f.Path, f.Mode); err != nil {
			return nil, fmt.Errorf("chmod %s: %w", f.Path, err)
		}
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Updated"}, nil
}

// diffContent produces a human-readable line-by-line diff, capped at 5 changes for readability.
func diffContent(old, new string) []extensions.Change {
	oldLines := strings.Split(strings.TrimRight(old, "\n"), "\n")
	newLines := strings.Split(strings.TrimRight(new, "\n"), "\n")

	var changes []extensions.Change

	maxLines := max(len(oldLines), len(newLines))
	shown := 0
	for i := range maxLines {
		if shown >= 5 {
			remaining := 0
			for j := i; j < maxLines; j++ {
				oldL, newL := lineAt(oldLines, j), lineAt(newLines, j)
				if oldL != newL {
					remaining++
				}
			}
			if remaining > 0 {
				changes = append(changes, extensions.Change{
					Property: "content", To: fmt.Sprintf("... and %d more lines", remaining), Action: "modify",
				})
			}
			break
		}
		oldL := lineAt(oldLines, i)
		newL := lineAt(newLines, i)
		if oldL == newL {
			continue
		}
		if oldL == "" {
			changes = append(changes, extensions.Change{
				Property: fmt.Sprintf("line %d", i+1), To: truncate(newL, 60), Action: "add",
			})
		} else if newL == "" {
			changes = append(changes, extensions.Change{
				Property: fmt.Sprintf("line %d", i+1), From: truncate(oldL, 60), Action: "remove",
			})
		} else {
			changes = append(changes, extensions.Change{
				Property: fmt.Sprintf("line %d", i+1), From: truncate(oldL, 40), To: truncate(newL, 40), Action: "modify",
			})
		}
		shown++
	}

	return changes
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

func summarizeContent(s string) string {
	s = strings.TrimRight(s, "\n\r")
	lines := strings.Count(s, "\n") + 1
	if lines == 1 {
		return truncate(s, 60)
	}
	return fmt.Sprintf("%d lines, %d bytes", lines, len(s))
}

func truncate(s string, maxLen int) string {
	s = strings.TrimRight(s, "\n\r")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
