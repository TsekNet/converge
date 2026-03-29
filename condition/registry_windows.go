//go:build windows

package condition

import (
	"context"
	"fmt"
	"strings"

	"github.com/TsekNet/converge/internal/winreg"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// RegistryKeyExists returns a Condition satisfied when the registry key exists.
func RegistryKeyExists(key string) *registryCondition {
	return &registryCondition{key: key}
}

// RegistryValueExists returns a Condition satisfied when the named value exists under key.
func RegistryValueExists(key, value string) *registryCondition {
	return &registryCondition{key: key, value: value}
}

// RegistryValueEquals returns a Condition satisfied when the named value's data equals the
// string representation of data (via fmt.Sprintf("%v", data)).
func RegistryValueEquals(key, value string, data any) *registryCondition {
	return &registryCondition{key: key, value: value, data: data, checkData: true}
}

// registryCondition is satisfied when a registry key or value reaches the desired state.
// Constructed via RegistryKeyExists, RegistryValueExists, or RegistryValueEquals.
type registryCondition struct {
	key       string
	value     string // empty = key-only check
	data      any    // compared via fmt.Sprintf("%v") when checkData is true
	checkData bool
}

func (c *registryCondition) Met(_ context.Context) (bool, error) {
	root, subkey, err := winreg.ParseKeyPath(c.key)
	if err != nil {
		return false, err
	}
	k, err := registry.OpenKey(root, subkey, registry.QUERY_VALUE)
	if err != nil {
		return false, nil //nolint:nilerr // key absent = not met
	}
	defer k.Close()

	if c.value == "" {
		return true, nil
	}
	if !c.checkData {
		_, _, err := k.GetValue(c.value, nil)
		return err == nil, nil
	}
	actual, err := readAsString(k, c.value)
	if err != nil {
		return false, nil //nolint:nilerr // value absent = not met
	}
	return actual == fmt.Sprintf("%v", c.data), nil
}

// Wait uses RegNotifyChangeKeyValue on the nearest existing ancestor to avoid
// polling. bWatchSubtree=true catches creations anywhere beneath the ancestor.
func (c *registryCondition) Wait(ctx context.Context) error {
	if met, err := c.Met(ctx); err != nil {
		return err
	} else if met {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		k, err := openNearestKey(c.key)
		if err != nil {
			return err
		}

		event, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			k.Close()
			return fmt.Errorf("CreateEvent: %w", err)
		}

		err = windows.RegNotifyChangeKeyValue(
			windows.Handle(k),
			true, // bWatchSubtree
			windows.REG_NOTIFY_CHANGE_NAME|windows.REG_NOTIFY_CHANGE_LAST_SET,
			event,
			true, // asynchronous
		)
		if err != nil {
			windows.CloseHandle(event)
			k.Close()
			return fmt.Errorf("RegNotifyChangeKeyValue: %w", err)
		}

		result, err := windows.WaitForSingleObject(event, 500)
		windows.CloseHandle(event)
		k.Close()
		if err != nil {
			return fmt.Errorf("WaitForSingleObject: %w", err)
		}
		if result == uint32(windows.WAIT_TIMEOUT) {
			if met, _ := c.Met(ctx); met {
				return nil
			}
			continue
		}

		if met, _ := c.Met(ctx); met {
			return nil
		}
	}
}

func (c *registryCondition) String() string {
	if c.value == "" {
		return fmt.Sprintf("registry key exists %s", c.key)
	}
	if !c.checkData {
		return fmt.Sprintf("registry value exists %s\\%s", c.key, c.value)
	}
	return fmt.Sprintf("registry value %s\\%s = %v", c.key, c.value, c.data)
}

// openNearestKey walks up the key path to find the nearest existing ancestor.
// Watching the ancestor with bWatchSubtree=true catches descendant creations.
func openNearestKey(full string) (registry.Key, error) {
	root, path, err := winreg.ParseKeyPath(full)
	if err != nil {
		return 0, err
	}
	parts := strings.Split(path, `\`)
	for i := len(parts); i > 0; i-- {
		candidate := strings.Join(parts[:i], `\`)
		k, err := registry.OpenKey(root, candidate, registry.NOTIFY)
		if err == nil {
			return k, nil
		}
	}
	return 0, fmt.Errorf("no existing ancestor found for %s", full)
}

// readAsString reads a registry value and returns its string representation.
// Integers are formatted with %d; strings are returned as-is.
func readAsString(k registry.Key, name string) (string, error) {
	s, _, err := k.GetStringValue(name)
	if err == nil {
		return s, nil
	}
	n, _, err := k.GetIntegerValue(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", n), nil
}
