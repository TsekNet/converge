package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type dnfManager struct{}

func (d *dnfManager) Name() string { return "dnf" }

func (d *dnfManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "dnf", "list", "installed", name)
	_, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (d *dnfManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "dnf", "install", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (d *dnfManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf remove %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (d *dnfManager) InstallBatch(ctx context.Context, names []string) error {
	args := append([]string{"install", "-y"}, names...)
	cmd := exec.CommandContext(ctx, "dnf", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf install %s: %s: %w", strings.Join(names, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (d *dnfManager) RemoveBatch(ctx context.Context, names []string) error {
	args := append([]string{"remove", "-y"}, names...)
	cmd := exec.CommandContext(ctx, "dnf", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf remove %s: %s: %w", strings.Join(names, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}
