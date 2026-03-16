//go:build windows

package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// Watch uses WMI event subscriptions to detect Windows service state changes
// in real-time via COM/IDispatch. Falls back to 5s polling if WMI fails.
func (s *Service) Watch(ctx context.Context, events chan<- extensions.Event) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		// COM init failed, fall back to polling.
		return s.pollFallback(ctx, events)
	}
	defer ole.CoUninitialize()

	locator, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return s.pollFallback(ctx, events)
	}
	defer locator.Release()

	wbem, err := locator.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return s.pollFallback(ctx, events)
	}
	defer wbem.Release()

	serviceResult, err := oleutil.CallMethod(wbem, "ConnectServer")
	if err != nil {
		return s.pollFallback(ctx, events)
	}
	svc := serviceResult.ToIDispatch()
	defer svc.Release()

	// WQL event query: fires when the service's State property changes.
	query := fmt.Sprintf(
		"SELECT * FROM __InstanceModificationEvent WITHIN 1 WHERE TargetInstance ISA 'Win32_Service' AND TargetInstance.Name = '%s'",
		s.Name,
	)

	eventSource, err := oleutil.CallMethod(svc, "ExecNotificationQuery", query)
	if err != nil {
		return s.pollFallback(ctx, events)
	}
	sink := eventSource.ToIDispatch()
	defer sink.Release()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// NextEvent with 1 second timeout (1000ms).
		// Returns the event object, or times out.
		result, err := oleutil.CallMethod(sink, "NextEvent", 1000)
		if err != nil {
			// Timeout (WBEM_E_TIMED_OUT = 0x80043001), just loop.
			continue
		}
		evt := result.ToIDispatch()
		evt.Release()

		select {
		case events <- extensions.Event{
			ResourceID: s.ID(),
			Kind:       extensions.EventWatch,
			Detail:     "WMI InstanceModificationEvent",
			Time:       time.Now(),
		}:
		case <-ctx.Done():
			return nil
		}
	}
}

// pollFallback is used when WMI event subscription fails.
func (s *Service) pollFallback(ctx context.Context, events chan<- extensions.Event) error {
	runtime.UnlockOSThread() // release thread lock from Watch

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			select {
			case events <- extensions.Event{
				ResourceID: s.ID(),
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
