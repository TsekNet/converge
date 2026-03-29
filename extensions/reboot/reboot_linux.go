//go:build linux

package reboot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/unix"
)

// bootTime reads /proc/uptime to calculate the approximate system boot time.
func bootTime() (time.Time, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}, fmt.Errorf("read /proc/uptime: %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return time.Time{}, fmt.Errorf("empty /proc/uptime")
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse /proc/uptime: %w", err)
	}
	return time.Now().Add(-time.Duration(secs * float64(time.Second))), nil
}

// Apply waits for the configured delay, writes the sentinel, then calls
// unix.Reboot. On success the kernel terminates the process, so the final
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
	if err := unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART); err != nil {
		return nil, fmt.Errorf("reboot: %w", err)
	}
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: r.effectiveMessage()}, nil
}
