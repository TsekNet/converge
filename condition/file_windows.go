//go:build windows

package condition

import (
	"context"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Wait uses ReadDirectoryChangesW on the parent directory to detect when
// the target file is created, then re-checks Met.
func (c *fileExistsCondition) Wait(ctx context.Context) error {
	dir := filepath.Dir(c.path)

	dirPtr, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}

	handle, err := windows.CreateFile(
		dirPtr,
		windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OVERLAPPED,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	// Check immediately before entering the wait loop.
	if met, _ := c.Met(ctx); met {
		return nil
	}

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var bytesReturned uint32
		overlapped := &windows.Overlapped{}
		overlapped.HEvent, err = windows.CreateEvent(nil, 1, 0, nil)
		if err != nil {
			return err
		}

		err = windows.ReadDirectoryChanges(
			handle,
			(*windows.FileNotifyInformation)(unsafe.Pointer(&buf[0])),
			uint32(len(buf)),
			false, // not subtree
			windows.FILE_NOTIFY_CHANGE_FILE_NAME,
			&bytesReturned,
			overlapped,
			0,
		)
		if err != nil {
			windows.CloseHandle(overlapped.HEvent)
			return err
		}

		// Wait for the event with a 500ms timeout to remain ctx-responsive.
		result, err := windows.WaitForSingleObject(overlapped.HEvent, 500)
		windows.CloseHandle(overlapped.HEvent)

		if err != nil {
			return err
		}
		if result == uint32(windows.WAIT_TIMEOUT) {
			// Recheck ctx; loop continues with a fresh ReadDirectoryChanges.
			continue
		}

		// Drain the overlapped result.
		var transferred uint32
		windows.GetOverlappedResult(handle, overlapped, &transferred, false)

		if met, _ := c.Met(ctx); met {
			return nil
		}

		// Small yield to avoid tight re-issue.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}
