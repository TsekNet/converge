//go:build darwin

package firewall

import (
	"fmt"
	"os/exec"
)

// pfctlReload reloads the pf configuration. macOS provides no stable
// userspace API for pf rule management, making pfctl the standard
// mechanism used by all tools (including Apple's own MDM profiles).
func pfctlReload() error {
	cmd := exec.Command("/sbin/pfctl", "-f", pfConf)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pfctl reload: %s: %w", out, err)
	}
	return nil
}
