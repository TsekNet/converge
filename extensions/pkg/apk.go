package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type apkManager struct{}

func (a *apkManager) Name() string { return "apk" }

func (a *apkManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "apk", "info", "-e", name)
	_, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (a *apkManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "apk", "add", "--no-cache", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apk add %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (a *apkManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "apk", "del", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apk del %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}
