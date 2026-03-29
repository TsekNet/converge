//go:build windows

package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/winreg"
	"golang.org/x/sys/windows/registry"
)

// Check opens the registry key read-only and compares the current value against desired.
func (r *Registry) Check(_ context.Context) (*extensions.State, error) {
	root, path, err := winreg.ParseKeyPath(r.Key)
	if err != nil {
		return nil, err
	}

	k, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		if r.State == "absent" {
			return &extensions.State{InSync: true}, nil
		}
		return &extensions.State{
			InSync:  false,
			Changes: []extensions.Change{{Property: r.Value, To: fmt.Sprintf("%v", r.Data), Action: "add"}},
		}, nil
	}
	defer k.Close()

	if r.State == "absent" {
		if _, _, err := k.GetValue(r.Value, nil); err != nil {
			return &extensions.State{InSync: true}, nil
		}
		return &extensions.State{
			InSync:  false,
			Changes: []extensions.Change{{Property: r.Value, From: "(exists)", To: "(absent)", Action: "remove"}},
		}, nil
	}

	current, err := r.readValue(k)
	if err != nil {
		return &extensions.State{
			InSync:  false,
			Changes: []extensions.Change{{Property: r.Value, To: fmt.Sprintf("%v", r.Data), Action: "add"}},
		}, nil
	}

	desired := fmt.Sprintf("%v", r.Data)
	if current != desired {
		return &extensions.State{
			InSync:  false,
			Changes: []extensions.Change{{Property: r.Value, From: current, To: desired, Action: "modify"}},
		}, nil
	}

	return &extensions.State{InSync: true}, nil
}

// Apply creates the key if needed and writes the value. For "absent" state, deletes the value.
func (r *Registry) Apply(_ context.Context) (*extensions.Result, error) {
	root, path, err := winreg.ParseKeyPath(r.Key)
	if err != nil {
		return nil, err
	}

	if r.State == "absent" {
		k, err := registry.OpenKey(root, path, registry.SET_VALUE)
		if err != nil {
			return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "key not found"}, nil
		}
		defer k.Close()
		if err := k.DeleteValue(r.Value); err != nil {
			return nil, fmt.Errorf("delete value %s\\%s: %w", r.Key, r.Value, err)
		}
		return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "deleted"}, nil
	}

	k, _, err := registry.CreateKey(root, path, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create key %s: %w", r.Key, err)
	}
	defer k.Close()

	if err := r.writeValue(k); err != nil {
		return nil, fmt.Errorf("set %s\\%s: %w", r.Key, r.Value, err)
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "set"}, nil
}

func (r *Registry) readValue(k registry.Key) (string, error) {
	switch normalizeType(r.Type) {
	case "dword":
		v, _, err := k.GetIntegerValue(r.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d", v), nil
	case "qword":
		v, _, err := k.GetIntegerValue(r.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d", v), nil
	case "multistring":
		v, _, err := k.GetStringsValue(r.Value)
		if err != nil {
			return "", err
		}
		return strings.Join(v, ","), nil
	case "binary":
		v, _, err := k.GetBinaryValue(r.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", v), nil
	default:
		v, _, err := k.GetStringValue(r.Value)
		if err != nil {
			return "", err
		}
		return v, nil
	}
}

func (r *Registry) writeValue(k registry.Key) error {
	switch normalizeType(r.Type) {
	case "dword":
		return k.SetDWordValue(r.Value, toUint32(r.Data))
	case "qword":
		return k.SetQWordValue(r.Value, toUint64(r.Data))
	case "expandstring":
		return k.SetExpandStringValue(r.Value, fmt.Sprintf("%v", r.Data))
	case "multistring":
		return k.SetStringsValue(r.Value, toStringSlice(r.Data))
	case "binary":
		return k.SetBinaryValue(r.Value, toBytes(r.Data))
	default:
		return k.SetStringValue(r.Value, fmt.Sprintf("%v", r.Data))
	}
}

// normalizeType accepts both Go-style ("dword") and Win32-style ("REG_DWORD") type names.
func normalizeType(t string) string {
	s := strings.ToLower(t)
	s = strings.TrimPrefix(s, "reg_")
	switch s {
	case "sz", "string", "":
		return "sz"
	case "expand_sz", "expandstring":
		return "expandstring"
	case "multi_sz", "multistring":
		return "multistring"
	default:
		return s
	}
}

func toUint32(v any) uint32 {
	switch n := v.(type) {
	case int:
		return uint32(n)
	case int64:
		return uint32(n)
	case uint32:
		return n
	case uint64:
		return uint32(n)
	case float64:
		return uint32(n)
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch n := v.(type) {
	case int:
		return uint64(n)
	case int64:
		return uint64(n)
	case uint32:
		return uint64(n)
	case uint64:
		return n
	case float64:
		return uint64(n)
	default:
		return 0
	}
}

func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case string:
		return strings.Split(s, ",")
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

func toBytes(v any) []byte {
	switch b := v.(type) {
	case []byte:
		return b
	case string:
		return []byte(b)
	default:
		return []byte(fmt.Sprintf("%v", v))
	}
}
