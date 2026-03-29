// Package reboot manages OS-level reboots as a convergent resource.
//
// Apply writes a sentinel file and schedules a platform-native reboot via OS
// APIs (no shell-outs). Check returns compliant only after the system has
// booted since Apply ran, determined by comparing boot time to the sentinel
// timestamp. The sentinel persists across reboots so re-convergence after a
// successful reboot is a fast no-op.
package reboot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
)

// Reboot schedules an OS reboot and tracks whether it has occurred.
// Check and Apply are implemented in platform-specific files.
type Reboot struct {
	Name     string
	Reason   string
	Message  string // optional user-facing message shown in converge output before the reboot fires
	Delay    time.Duration
	Critical bool

	// sentinelOverride, when non-empty, overrides the default sentinel path.
	// Used in tests to point at t.TempDir() instead of the system directory.
	sentinelOverride string
}

// New returns a Reboot with the given name. Path separators are stripped
// to prevent sentinel file path traversal; empty or dot-only names are
// rejected by the DSL require() check.
func New(name string) *Reboot {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	if name == "." {
		name = ""
	}
	return &Reboot{Name: name}
}

// effectiveMessage returns Message if set, otherwise Reason.
func (r *Reboot) effectiveMessage() string {
	if r.Message != "" {
		return r.Message
	}
	return r.Reason
}

func (r *Reboot) ID() string       { return "reboot:" + r.Name }
func (r *Reboot) String() string   { return "Reboot " + r.Name }
func (r *Reboot) IsCritical() bool { return r.Critical }

// sentinelPath returns the file written by Apply to record when the reboot
// was requested. Check compares this timestamp against the current boot time.
func (r *Reboot) sentinelPath() string {
	if r.sentinelOverride != "" {
		return r.sentinelOverride
	}
	return filepath.Join(sentinelDir(), fmt.Sprintf("reboot-%s.sentinel", r.Name))
}

func sentinelDir() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\ProgramData\converge`
	default:
		return "/var/lib/converge"
	}
}

// writeSentinel creates the sentinel directory and writes the current unix
// timestamp truncated to seconds. Boot time sources (GetTickCount64,
// /proc/uptime, kern.boottime) have at best millisecond precision, so
// nanosecond sentinels would create false-negative comparisons.
func writeSentinel(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create sentinel dir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0o644)
}

// sentinelTime reads the timestamp written by writeSentinel. Supports both
// second-precision (current) and legacy nanosecond-precision sentinels.
// Returns zero time if the sentinel does not exist.
func sentinelTime(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse sentinel %s: %w", path, err)
	}
	// Distinguish second vs nanosecond timestamps: anything above 1e12
	// (year 33658) is a nanosecond timestamp from a legacy sentinel.
	if v > 1e12 {
		return time.Unix(0, v), nil
	}
	return time.Unix(v, 0), nil
}

// removeSentinel deletes the sentinel file. Used to recover from cancelled
// reboots that would otherwise leave the resource permanently drifted.
func (r *Reboot) removeSentinel() error {
	err := os.Remove(r.sentinelPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// sentinelGrace is the maximum age of a sentinel whose corresponding reboot
// never completed. After this duration, Apply removes the stale sentinel
// before writing a new one, recovering from cancelled reboots.
const sentinelGrace = 10 * time.Minute

// Check is the platform-agnostic Check implementation. Each platform file
// provides only bootTime(); Check delegates to checkState for the comparison.
func (r *Reboot) Check(_ context.Context) (*extensions.State, error) {
	bt, err := bootTime()
	if err != nil {
		return nil, fmt.Errorf("boot time: %w", err)
	}
	return r.checkState(bt)
}

// checkState is the platform-agnostic Check logic. bt is the current boot time.
//
//   - No sentinel: Apply has never run; reboot not yet requested, drifted.
//   - Sentinel exists, boot after sentinel: reboot completed, compliant.
//   - Sentinel exists, boot before sentinel: reboot requested but pending, drifted.
//   - Sentinel older than sentinelGrace and no reboot occurred: stale, drifted.
//
// A 2-second grace window accounts for boot time measurement imprecision
// across platforms (GetTickCount64 ms, /proc/uptime sub-second, kern.boottime us).
//
// Check is read-only: stale sentinel cleanup is deferred to Apply.
func (r *Reboot) checkState(bt time.Time) (*extensions.State, error) {
	st, err := sentinelTime(r.sentinelPath())
	if err != nil {
		return nil, err
	}
	if st.IsZero() {
		return &extensions.State{
			InSync:  false,
			Changes: []extensions.Change{{Property: "reboot", To: "pending", Action: "reboot"}},
		}, nil
	}
	if bt.After(st.Add(-2 * time.Second)) {
		return &extensions.State{InSync: true}, nil
	}
	// Sentinel exists but the machine has not rebooted. Report drifted.
	// If the sentinel is stale (older than sentinelGrace), Apply will
	// clean it up before re-requesting the reboot.
	return &extensions.State{
		InSync:  false,
		Changes: []extensions.Change{{Property: "reboot", From: "requested", To: "completed", Action: "reboot"}},
	}, nil
}
