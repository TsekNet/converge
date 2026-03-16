//go:build windows

package service

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	advapi32                      = windows.NewLazySystemDLL("advapi32.dll")
	procNotifyServiceStatusChange = advapi32.NewProc("NotifyServiceStatusChangeW")
)

// SERVICE_NOTIFY constants.
const (
	serviceNotifyStatusChange   = 0x00000002
	serviceNotifyStopped        = 0x00000001
	serviceNotifyStartPending   = 0x00000002
	serviceNotifyStopPending    = 0x00000004
	serviceNotifyRunning        = 0x00000008
	serviceNotifyPaused         = 0x00000020
	serviceNotifyPausePending   = 0x00000040
	serviceNotifyContinuePending = 0x00000010
	serviceNotifyDeletePending  = 0x00000200
	serviceNotifyAll            = 0x000003FF
)

// serviceNotify2 mirrors the SERVICE_NOTIFY_2 structure.
type serviceNotify2 struct {
	Version               uint32
	NotifyCallback        uintptr
	Context               uintptr
	NotificationStatus    uint32
	ServiceStatus         windows.SERVICE_STATUS_PROCESS
	NotificationTriggered uint32
	ServiceNames          *uint16
}

// Watch uses NotifyServiceStatusChangeW to detect service state changes
// on Windows via the native SCM API.
func (s *Service) Watch(ctx context.Context, events chan<- extensions.Event) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect SCM: %w", err)
	}
	defer m.Disconnect()

	svc, err := m.OpenService(s.Name)
	if err != nil {
		return fmt.Errorf("open service %s: %w", s.Name, err)
	}
	defer svc.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		notify := &serviceNotify2{
			Version: serviceNotifyStatusChange,
		}

		r, _, err := procNotifyServiceStatusChange.Call(
			uintptr(svc.Handle),
			uintptr(serviceNotifyAll),
			uintptr(unsafe.Pointer(notify)),
		)
		if r != 0 {
			// Fallback to polling if NotifyServiceStatusChange fails.
			time.Sleep(5 * time.Second)
			select {
			case events <- extensions.Event{
				ResourceID: s.ID(),
				Reason:     "poll (SCM notify unavailable)",
				Time:       time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
			continue
		}
		_ = err

		// Wait for the notification to fire (SleepEx with alertable=true).
		windows.SleepEx(500, true)

		if notify.NotificationTriggered != 0 {
			select {
			case events <- extensions.Event{
				ResourceID: s.ID(),
				Reason:     "NotifyServiceStatusChange",
				Time:       time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
