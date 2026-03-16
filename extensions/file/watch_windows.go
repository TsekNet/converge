//go:build windows

package file

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
)

// Watch uses ReadDirectoryChangesW to monitor the file's parent directory
// for changes on Windows.
func (f *File) Watch(ctx context.Context, events chan<- extensions.Event) error {
	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	dir := filepath.Dir(absPath)
	dirPtr, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return fmt.Errorf("utf16 dir: %w", err)
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
		return fmt.Errorf("CreateFile %s: %w", dir, err)
	}
	defer windows.CloseHandle(handle)

	const bufSize = 4096
	buf := make([]byte, bufSize)

	overlap := &windows.Overlapped{}
	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("CreateEvent: %w", err)
	}
	defer windows.CloseHandle(event)
	overlap.HEvent = event

	filter := uint32(windows.FILE_NOTIFY_CHANGE_FILE_NAME |
		windows.FILE_NOTIFY_CHANGE_DIR_NAME |
		windows.FILE_NOTIFY_CHANGE_ATTRIBUTES |
		windows.FILE_NOTIFY_CHANGE_SIZE |
		windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
		windows.FILE_NOTIFY_CHANGE_CREATION)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err = windows.ReadDirectoryChanges(
			handle,
			&buf[0],
			uint32(bufSize),
			false, // watch tree
			filter,
			nil,
			overlap,
			0,
		)
		if err != nil {
			return fmt.Errorf("ReadDirectoryChanges: %w", err)
		}

		// Wait with timeout so we can check ctx.Done().
		r, err := windows.WaitForSingleObject(event, 500)
		if err != nil {
			return fmt.Errorf("WaitForSingleObject: %w", err)
		}
		if r == uint32(windows.WAIT_TIMEOUT) {
			continue
		}

		var bytesReturned uint32
		windows.GetOverlappedResult(handle, overlap, &bytesReturned, false)

		if bytesReturned > 0 {
			// Parse FILE_NOTIFY_INFORMATION to check if our target file changed.
			offset := uint32(0)
			for {
				info := (*fileNotifyInformation)(unsafe.Pointer(&buf[offset]))
				nameLen := info.FileNameLength / 2
				name := windows.UTF16ToString((*[1 << 15]uint16)(unsafe.Pointer(&info.FileName))[:nameLen:nameLen])
				if filepath.Base(absPath) == name {
					select {
					case events <- extensions.Event{
						ResourceID: f.ID(),
						Reason:     "ReadDirectoryChangesW",
						Time:       time.Now(),
					}:
					case <-ctx.Done():
						return nil
					}
					break
				}
				if info.NextEntryOffset == 0 {
					break
				}
				offset += info.NextEntryOffset
			}
		}
	}
}

// fileNotifyInformation mirrors the Windows FILE_NOTIFY_INFORMATION structure.
type fileNotifyInformation struct {
	NextEntryOffset uint32
	Action          uint32
	FileNameLength  uint32
	FileName        uint16
}
