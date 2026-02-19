package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type aptManager struct{}

func (a *aptManager) Name() string { return "apt" }

func (a *aptManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Status}", name)
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.Contains(string(out), "install ok installed"), nil
}

func (a *aptManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "apt-get", "install", "-y", "--no-install-recommends", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt-get install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (a *aptManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "apt-get", "remove", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt-get remove %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}
