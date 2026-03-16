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
	cmd := exec.CommandContext(ctx, "winget", "list", "--id", name, "--exact", "--accept-source-agreements")
	out, _ := cmd.CombinedOutput()
	// winget list outputs a table with the package ID if installed,
	// or "No installed package found" if not. Check for the ID in the output
	// regardless of exit code (winget exit codes are inconsistent).
	output := string(out)
	if strings.Contains(output, name) && !strings.Contains(output, "No installed package found") {
		return true, nil
	}
	return false, nil
}

func (w *wingetManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "winget", "install", "--id", name, "--exact",
		"--accept-package-agreements", "--accept-source-agreements", "--silent")
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		// winget returns non-zero even on "already installed" sometimes.
		if strings.Contains(output, "already installed") {
			return nil
		}
		return fmt.Errorf("winget install %s: %s: %w", name, output, err)
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
