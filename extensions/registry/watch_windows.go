//go:build windows

package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
	winreg "golang.org/x/sys/windows/registry"
)

// Watch uses RegNotifyChangeKeyValue to monitor the registry key for
// value changes on Windows.
func (r *Registry) Watch(ctx context.Context, events chan<- extensions.Event) error {
	root, subkey, err := parseKeyPath(r.Key)
	if err != nil {
		return err
	}

	key, err := winreg.OpenKey(root, subkey, winreg.NOTIFY)
	if err != nil {
		return fmt.Errorf("open registry key %s: %w", r.Key, err)
	}
	defer key.Close()

	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("CreateEvent: %w", err)
	}
	defer windows.CloseHandle(event)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// REG_NOTIFY_CHANGE_LAST_SET = value changes.
		err = windows.RegNotifyChangeKeyValue(
			windows.Handle(key),
			false,
			windows.REG_NOTIFY_CHANGE_LAST_SET,
			event,
			true, // async
		)
		if err != nil {
			return fmt.Errorf("RegNotifyChangeKeyValue: %w", err)
		}

		result, err := windows.WaitForSingleObject(event, 500)
		if err != nil {
			return fmt.Errorf("WaitForSingleObject: %w", err)
		}
		if result == uint32(windows.WAIT_TIMEOUT) {
			continue
		}

		select {
		case events <- extensions.Event{
			ResourceID: r.ID(),
			Reason:     "RegNotifyChangeKeyValue",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}

// parseKeyPath splits "HKLM\SOFTWARE\...\ValueName" into root key and subkey.
func parseKeyPath(path string) (winreg.Key, string, error) {
	parts := strings.SplitN(path, `\`, 2)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid registry path: %s", path)
	}

	var root winreg.Key
	switch strings.ToUpper(parts[0]) {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		root = winreg.LOCAL_MACHINE
	case "HKCU", "HKEY_CURRENT_USER":
		root = winreg.CURRENT_USER
	case "HKCR", "HKEY_CLASSES_ROOT":
		root = winreg.CLASSES_ROOT
	case "HKU", "HKEY_USERS":
		root = winreg.USERS
	default:
		return 0, "", fmt.Errorf("unknown registry root: %s", parts[0])
	}

	return root, parts[1], nil
}
