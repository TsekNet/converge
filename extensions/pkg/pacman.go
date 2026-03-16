package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type pacmanManager struct{}

func (p *pacmanManager) Name() string { return "pacman" }

func (p *pacmanManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "pacman", "-Q", name)
	_, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (p *pacmanManager) Install(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "pacman", "-S", "--noconfirm", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman -S %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (p *pacmanManager) Remove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "pacman", "-R", "--noconfirm", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman -R %s: %s: %w", name, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (p *pacmanManager) InstallBatch(ctx context.Context, names []string) error {
	args := append([]string{"-S", "--noconfirm"}, names...)
	cmd := exec.CommandContext(ctx, "pacman", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman -S %s: %s: %w", strings.Join(names, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (p *pacmanManager) RemoveBatch(ctx context.Context, names []string) error {
	args := append([]string{"-R", "--noconfirm"}, names...)
	cmd := exec.CommandContext(ctx, "pacman", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman -R %s: %s: %w", strings.Join(names, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}
