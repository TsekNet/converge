package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type wingetManager struct{}

func (w *wingetManager) Name() string { return "winget" }

func (w *wingetManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "winget", "list", "--exact", "-q", name)
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(name)), nil
}

func (w *wingetManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "winget", "install", "--exact", "--accept-package-agreements", "--accept-source-agreements", "-h", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("winget install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (w *wingetManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "winget", "uninstall", "--exact", "-h", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("winget uninstall %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (w *wingetManager) InstallBatch(ctx context.Context, names []string) error {
	for _, name := range names {
		if err := w.Install(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (w *wingetManager) RemoveBatch(ctx context.Context, names []string) error {
	for _, name := range names {
		if err := w.Remove(ctx, name); err != nil {
			return err
		}
	}
	return nil
}
