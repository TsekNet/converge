//go:build windows

package condition

import (
	"context"
	"path/filepath"

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

	// Create the event handle once; reuse it across ReadDirectoryChanges calls.
	eventHandle, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(eventHandle)

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var bytesReturned uint32
		overlapped := &windows.Overlapped{HEvent: eventHandle}

		if err := windows.ReadDirectoryChanges(
			handle,
			&buf[0],
			uint32(len(buf)),
			false,
			windows.FILE_NOTIFY_CHANGE_FILE_NAME,
			&bytesReturned,
			overlapped,
			0,
		); err != nil {
			return err
		}

		// Wait for the event with a 500ms timeout to remain ctx-responsive.
		result, err := windows.WaitForSingleObject(eventHandle, 500)
		if err != nil {
			return err
		}
		if result == uint32(windows.WAIT_TIMEOUT) {
			// Cancel the pending I/O before reissuing ReadDirectoryChanges.
			windows.CancelIoEx(handle, overlapped)
			continue
		}

		var transferred uint32
		if err := windows.GetOverlappedResult(handle, overlapped, &transferred, false); err != nil {
			return err
		}

		if met, _ := c.Met(ctx); met {
			return nil
		}
	}
}
