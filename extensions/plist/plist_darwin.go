//go:build darwin

package plist

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TsekNet/converge/extensions"
	goplist "howett.net/plist"
)

func (p *Plist) Check(_ context.Context) (*extensions.State, error) {
	current, err := p.readKey()
	if err != nil {
		return &extensions.State{
			InSync: false,
			Changes: []extensions.Change{{
				Property: p.Key,
				To:       fmt.Sprintf("%v", p.Value),
				Action:   "add",
			}},
		}, nil
	}

	if fmt.Sprintf("%v", current) == fmt.Sprintf("%v", p.Value) {
		return &extensions.State{InSync: true}, nil
	}

	return &extensions.State{
		InSync: false,
		Changes: []extensions.Change{{
			Property: p.Key,
			From:     fmt.Sprintf("%v", current),
			To:       fmt.Sprintf("%v", p.Value),
			Action:   "modify",
		}},
	}, nil
}

func (p *Plist) Apply(_ context.Context) (*extensions.Result, error) {
	path := p.plistPath()
	data := make(map[string]any)

	if raw, err := os.ReadFile(path); err == nil {
		if _, decErr := goplist.Unmarshal(raw, &data); decErr != nil {
			data = make(map[string]any)
		}
	}

	data[p.Key] = p.Value

	out, err := goplist.Marshal(data, goplist.BinaryFormat)
	if err != nil {
		return nil, fmt.Errorf("marshal plist %s: %w", p.Domain, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return nil, fmt.Errorf("write plist %s: %w", path, err)
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "set"}, nil
}

func (p *Plist) readKey() (any, error) {
	raw, err := os.ReadFile(p.plistPath())
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if _, err := goplist.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	val, ok := data[p.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in %s", p.Key, p.Domain)
	}
	return val, nil
}

func (p *Plist) plistPath() string {
	if p.Host {
		return filepath.Join("/Library/Preferences", p.Domain+".plist")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Preferences", p.Domain+".plist")
}
