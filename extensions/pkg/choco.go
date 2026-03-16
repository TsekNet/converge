package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type chocoManager struct{}

func (c *chocoManager) Name() string { return "choco" }

func (c *chocoManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "choco", "list", "--local-only", "--exact", name)
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(name)), nil
}

func (c *chocoManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "choco", "install", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("choco install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (c *chocoManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "choco", "uninstall", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("choco uninstall %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (c *chocoManager) InstallBatch(ctx context.Context, names []string) error {
	args := append([]string{"install", "-y"}, names...)
	cmd := exec.CommandContext(ctx, "choco", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("choco install %s: %s: %w", strings.Join(names, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (c *chocoManager) RemoveBatch(ctx context.Context, names []string) error {
	args := append([]string{"uninstall", "-y"}, names...)
	cmd := exec.CommandContext(ctx, "choco", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("choco uninstall %s: %s: %w", strings.Join(names, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}
