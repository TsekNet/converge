//go:build windows

package reboot

import (
	"context"
	"fmt"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
)

var (
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	advapi32         = windows.NewLazySystemDLL("advapi32.dll")
	procGetTick      = kernel32.NewProc("GetTickCount64")
	procInitShutdown = advapi32.NewProc("InitiateSystemShutdownExW")
)

// bootTime returns the approximate system boot time via GetTickCount64.
// GetTickCount64 is milliseconds since boot; no WMI or shell required.
// The return value is cast to uint64 to avoid truncation on 32-bit builds
// where uintptr is 32 bits but GetTickCount64 returns a 64-bit value.
// On amd64, r1 holds the full value; the r2 shift is only needed on 32-bit builds.
func bootTime() (time.Time, error) {
	r1, r2, _ := procGetTick.Call()
	ms := uint64(r1) | (uint64(r2) << 32)
	return time.Now().Add(-time.Duration(ms) * time.Millisecond), nil
}

// Apply acquires SE_SHUTDOWN_PRIVILEGE and calls InitiateSystemShutdownExW.
// A minimum 1-second delay ensures Apply returns before the OS fires the reboot.
// ctx is checked before starting but not during the delay: the OS manages the
// countdown asynchronously via dwTimeout. Cancellation after Apply returns
// requires AbortSystemShutdown.
func (r *Reboot) Apply(ctx context.Context) (*extensions.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := writeSentinel(r.sentinelPath()); err != nil {
		return nil, fmt.Errorf("write sentinel for %s: %w", r.ID(), err)
	}
	if err := acquireShutdownPrivilege(); err != nil {
		return nil, fmt.Errorf("acquire shutdown privilege: %w", err)
	}

	delay := uint32(r.Delay.Seconds())
	if delay == 0 {
		delay = 1 // minimum: let Apply return before the reboot fires
	}

	const (
		flagMajorApp   = 0x00040000
		flagMinorMaint = 0x00000002
		flagPlanned    = 0x80000000
	)
	ret, _, err := procInitShutdown.Call(
		0,              // lpMachineName: NULL = local machine
		0,              // lpMessage: NULL; message is logged via Result, not the shutdown dialog
		uintptr(delay), // dwTimeout in seconds
		1,              // bForceAppsClosed
		1,              // bRebootAfterShutdown
		uintptr(flagMajorApp|flagMinorMaint|flagPlanned),
	)
	if ret == 0 {
		return nil, fmt.Errorf("InitiateSystemShutdownExW: %w", err)
	}
	return &extensions.Result{
		Changed: true,
		Status:  extensions.StatusChanged,
		Message: fmt.Sprintf("reboot scheduled in %ds: %s", delay, r.effectiveMessage()),
	}, nil
}

// acquireShutdownPrivilege enables SE_SHUTDOWN_PRIVILEGE on the current
// process token. The privilege persists for the lifetime of the process.
func acquireShutdownPrivilege() error {
	var token windows.Token
	if err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY,
		&token,
	); err != nil {
		return err
	}
	defer token.Close()

	var luid windows.LUID
	if err := windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr("SeShutdownPrivilege"), &luid); err != nil {
		return err
	}

	privs := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges: [1]windows.LUIDAndAttributes{
			{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED},
		},
	}
	if err := windows.AdjustTokenPrivileges(token, false, &privs, 0, nil, nil); err != nil {
		return err
	}
	// AdjustTokenPrivileges returns nil even when ERROR_NOT_ALL_ASSIGNED;
	// check GetLastError to detect partial privilege application.
	if errno := windows.GetLastError(); errno != windows.ERROR_SUCCESS {
		return fmt.Errorf("AdjustTokenPrivileges: not all privileges assigned: %w", errno)
	}
	return nil
}
