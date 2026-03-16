//go:build linux || darwin

package condition

import (
	"context"
	"path/filepath"
	"syscall"
)

type mountPointCondition struct {
	path string
}

// Met returns true when path is on a different device than its parent,
// indicating it is a mount point.
func (c *mountPointCondition) Met(_ context.Context) (bool, error) {
	var stat, parentStat syscall.Stat_t
	if err := syscall.Stat(c.path, &stat); err != nil {
		return false, nil //nolint:nilerr // not present = not met
	}
	parent := filepath.Dir(c.path)
	if err := syscall.Stat(parent, &parentStat); err != nil {
		return false, err
	}
	return stat.Dev != parentStat.Dev, nil
}

func (c *mountPointCondition) String() string {
	return "mount point " + c.path
}
