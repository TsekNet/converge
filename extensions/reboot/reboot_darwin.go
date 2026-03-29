//go:build darwin

package reboot

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// Darwin reboot constants from <sys/reboot.h>. Not exported by x/sys/unix.
const rbAutoboot = 0 // RB_AUTOBOOT

// bootTime reads kern.boottime via sysctl to get the exact system boot time.
func bootTime() (time.Time, error) {
	tv, err := unix.SysctlTimeval("kern.boottime")
	if err != nil {
		return time.Time{}, fmt.Errorf("kern.boottime: %w", err)
	}
	return time.Unix(tv.Sec, int64(tv.Usec)*1000), nil
}

// Apply waits for the configured delay, writes the sentinel, then calls
// reboot(2). On success the kernel terminates the process, so the final
// return is only reached if Reboot fails.
func (r *Reboot) Apply(ctx context.Context) (*extensions.Result, error) {
	if r.Delay > 0 {
		select {
		case <-time.After(r.Delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err := writeSentinel(r.sentinelPath()); err != nil {
		return nil, fmt.Errorf("write sentinel for %s: %w", r.ID(), err)
	}
	if err := rebootDarwin(); err != nil {
		return nil, fmt.Errorf("reboot: %w", err)
	}
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: r.effectiveMessage()}, nil
}

// rebootDarwin calls the reboot(2) syscall directly because golang.org/x/sys/unix
// does not export Reboot or RB_AUTOBOOT for Darwin.
func rebootDarwin() error {
	_, _, errno := syscall.Syscall(syscall.SYS_REBOOT, uintptr(rbAutoboot), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
