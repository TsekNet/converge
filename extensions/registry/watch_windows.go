//go:build windows

package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/TsekNet/converge/extensions"
	cwinreg "github.com/TsekNet/converge/internal/winreg"
	"golang.org/x/sys/windows"
	winreg "golang.org/x/sys/windows/registry"
)

// Watch uses RegNotifyChangeKeyValue to monitor the registry key for
// value changes on Windows.
func (r *Registry) Watch(ctx context.Context, events chan<- extensions.Event) error {
	root, subkey, err := cwinreg.ParseKeyPath(r.Key)
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

		err = windows.RegNotifyChangeKeyValue(
			windows.Handle(key),
			false,
			windows.REG_NOTIFY_CHANGE_LAST_SET,
			event,
			true,
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

		// Re-register BEFORE sending the event to minimize the monitoring gap.
		err = windows.RegNotifyChangeKeyValue(
			windows.Handle(key),
			false,
			windows.REG_NOTIFY_CHANGE_LAST_SET,
			event,
			true,
		)
		if err != nil {
			return fmt.Errorf("RegNotifyChangeKeyValue (re-register): %w", err)
		}

		select {
		case events <- extensions.Event{
			ResourceID: r.ID(),
			Kind:       extensions.EventWatch,
			Detail:     "RegNotifyChangeKeyValue",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}
