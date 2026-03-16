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
	// Try exact ID match first, then fall back to name match.
	cmd := exec.CommandContext(ctx, "winget", "list", "--id", name, "--exact", "--accept-source-agreements")
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(name)) {
		return true, nil
	}

	// Fall back to name search.
	cmd = exec.CommandContext(ctx, "winget", "list", "--name", name, "--exact", "--accept-source-agreements")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(name)), nil
}

func (w *wingetManager) Install(ctx context.Context, name string) error {
	// Use --id for exact match, avoiding "multiple packages found" errors.
	cmd := exec.CommandContext(ctx, "winget", "install", "--id", name, "--exact",
		"--accept-package-agreements", "--accept-source-agreements", "--silent")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("winget install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (w *wingetManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "winget", "uninstall", "--id", name, "--exact", "--silent")
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
