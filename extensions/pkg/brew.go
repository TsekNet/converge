package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type brewManager struct{}

func (b *brewManager) Name() string { return "brew" }

func (b *brewManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "brew", "list", "--formula", name)
	err := cmd.Run()
	return err == nil, nil
}

func (b *brewManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "brew", "install", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (b *brewManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "brew", "uninstall", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew uninstall %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}
