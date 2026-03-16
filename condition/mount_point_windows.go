//go:build windows

package condition

import (
	"context"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type mountPointCondition struct {
	path string
}

// Met uses GetVolumePathName to determine whether path is the root of a volume.
func (c *mountPointCondition) Met(_ context.Context) (bool, error) {
	absPath, err := filepath.Abs(c.path)
	if err != nil {
		return false, err
	}

	pathPtr, err := windows.UTF16PtrFromString(absPath)
	if err != nil {
		return false, err
	}

	buf := make([]uint16, windows.MAX_PATH)
	err = windows.GetVolumePathName(pathPtr, &buf[0], uint32(len(buf)))
	if err != nil {
		return false, nil //nolint:nilerr // path not accessible = not met
	}

	volumeRoot := windows.UTF16ToString(buf)
	// Normalise: GetVolumePathName returns e.g. "C:\" for any path on C:.
	// A path is a mount point if it IS the volume root or a junction point.
	absNorm := strings.TrimRight(absPath, `\/`) + `\`
	volNorm := strings.TrimRight(volumeRoot, `\/`) + `\`
	if strings.EqualFold(absNorm, volNorm) {
		return true, nil
	}

	// Also check for junction/reparse points, which Windows uses for mount points.
	return isReparsePoint(absPath), nil
}

func isReparsePoint(path string) bool {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, err := windows.GetFileAttributes(p)
	if err != nil {
		return false
	}
	return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

// Wait uses RegisterDeviceNotification for volume arrival events.
// This requires a window message loop, which is complex. The pragmatic
// fallback is a 5-second poll: Windows mount events are infrequent enough
// that polling is acceptable here.
func (c *mountPointCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if met, _ := c.Met(ctx); met {
				return nil
			}
		}
	}
}

func (c *mountPointCondition) String() string {
	return "mount point " + c.path
}

// Suppress unused import warning; windows.MAX_PATH is used above.
var _ = unsafe.Pointer(nil)
