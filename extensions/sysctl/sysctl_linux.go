//go:build linux

package sysctl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

const procSysBase = "/proc/sys"

// Check reads the live kernel value from /proc/sys/<key> and compares it.
func (s *Sysctl) Check(_ context.Context) (*extensions.State, error) {
	current, err := s.read()
	if err != nil {
		return nil, fmt.Errorf("read sysctl %s: %w", s.Key, err)
	}

	if current == s.Value {
		return &extensions.State{InSync: true}, nil
	}

	return &extensions.State{
		InSync: false,
		Changes: []extensions.Change{{
			Property: s.Key,
			From:     current,
			To:       s.Value,
			Action:   "modify",
		}},
	}, nil
}

// Apply writes the value to /proc/sys/ for immediate effect, then optionally
// persists it to /etc/sysctl.d/99-converge.conf so it survives reboots.
func (s *Sysctl) Apply(_ context.Context) (*extensions.Result, error) {
	p := keyToPath(s.Key)
	if err := os.WriteFile(p, []byte(s.Value+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("write sysctl %s: %w", s.Key, err)
	}

	if s.Persist {
		if err := s.writePersist(); err != nil {
			return nil, fmt.Errorf("persist sysctl %s: %w", s.Key, err)
		}
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "set"}, nil
}

func (s *Sysctl) read() (string, error) {
	data, err := os.ReadFile(keyToPath(s.Key))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (s *Sysctl) writePersist() error {
	dir := "/etc/sysctl.d"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	line := fmt.Sprintf("%s = %s\n", s.Key, s.Value)
	return os.WriteFile(filepath.Join(dir, "99-converge.conf"), []byte(line), 0644)
}

// keyToPath converts "net.ipv4.ip_forward" to "/proc/sys/net/ipv4/ip_forward".
func keyToPath(key string) string {
	return filepath.Join(procSysBase, strings.ReplaceAll(key, ".", "/"))
}
