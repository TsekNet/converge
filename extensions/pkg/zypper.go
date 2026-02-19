package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type zypperManager struct{}

func (z *zypperManager) Name() string { return "zypper" }

func (z *zypperManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "zypper", "se", "--installed-only", "--match-exact", name)
	_, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (z *zypperManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "zypper", "install", "-n", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zypper install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (z *zypperManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "zypper", "remove", "-n", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zypper remove %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}
