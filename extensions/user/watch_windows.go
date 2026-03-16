//go:build windows

package user

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// Watch uses WMI event subscriptions to detect user account changes
// in real-time via COM/IDispatch.
func (u *User) Watch(ctx context.Context, events chan<- extensions.Event) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return u.pollFallback(ctx, events)
	}
	defer ole.CoUninitialize()

	locator, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return u.pollFallback(ctx, events)
	}
	defer locator.Release()

	wbem, err := locator.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return u.pollFallback(ctx, events)
	}
	defer wbem.Release()

	svcResult, err := oleutil.CallMethod(wbem, "ConnectServer")
	if err != nil {
		return u.pollFallback(ctx, events)
	}
	svc := svcResult.ToIDispatch()
	defer svc.Release()

	// Watch for any modification to the user account.
	query := fmt.Sprintf(
		"SELECT * FROM __InstanceModificationEvent WITHIN 2 WHERE TargetInstance ISA 'Win32_UserAccount' AND TargetInstance.Name = '%s'",
		u.Name,
	)

	eventSource, err := oleutil.CallMethod(svc, "ExecNotificationQuery", query)
	if err != nil {
		return u.pollFallback(ctx, events)
	}
	sink := eventSource.ToIDispatch()
	defer sink.Release()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		result, err := oleutil.CallMethod(sink, "NextEvent", 1000)
		if err != nil {
			continue // timeout, loop
		}
		evt := result.ToIDispatch()
		evt.Release()

		select {
		case events <- extensions.Event{
			ResourceID: u.ID(),
			Kind:       extensions.EventWatch,
			Detail:     "WMI Win32_UserAccount",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}

func (u *User) pollFallback(ctx context.Context, events chan<- extensions.Event) error {
	runtime.UnlockOSThread()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			select {
			case events <- extensions.Event{
				ResourceID: u.ID(),
				Kind:       extensions.EventPoll,
				Detail:     "WMI unavailable, polling",
				Time:       time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
