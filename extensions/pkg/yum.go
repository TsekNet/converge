package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type yumManager struct{}

func (y *yumManager) Name() string { return "yum" }

func (y *yumManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "yum", "list", "installed", name)
	_, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (y *yumManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "yum", "install", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yum install %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (y *yumManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "yum", "remove", "-y", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yum remove %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}
