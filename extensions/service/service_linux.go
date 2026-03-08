//go:build linux

package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

func (s *Service) Check(ctx context.Context) (*extensions.State, error) {
	switch s.InitSystem {
	case "systemd":
		return s.checkSystemd(ctx)
	default:
		return nil, fmt.Errorf("unsupported init system: %q", s.InitSystem)
	}
}

func (s *Service) Apply(ctx context.Context) (*extensions.Result, error) {
	switch s.InitSystem {
	case "systemd":
		return s.applySystemd(ctx)
	default:
		return nil, fmt.Errorf("apply not implemented for init system: %q", s.InitSystem)
	}
}

func (s *Service) checkSystemd(ctx context.Context) (*extensions.State, error) {
	var changes []extensions.Change

	activeCmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", s.Name)
	isActive := activeCmd.Run() == nil

	wantRunning := s.State == "running"
	if isActive != wantRunning {
		from, to := "running", "stopped"
		if wantRunning {
			from, to = "stopped", "running"
		}
		changes = append(changes, extensions.Change{
			Property: "state", From: from, To: to, Action: "modify",
		})
	}

	enabledCmd := exec.CommandContext(ctx, "systemctl", "is-enabled", "--quiet", s.Name)
	isEnabled := enabledCmd.Run() == nil

	if s.Enable && !isEnabled {
		changes = append(changes, extensions.Change{
			Property: "enabled", From: "false", To: "true", Action: "modify",
		})
	} else if !s.Enable && isEnabled {
		changes = append(changes, extensions.Change{
			Property: "enabled", From: "true", To: "false", Action: "modify",
		})
	}

	return &extensions.State{InSync: len(changes) == 0, Changes: changes}, nil
}

func (s *Service) applySystemd(ctx context.Context) (*extensions.Result, error) {
	if s.State == "running" {
		if out, err := exec.CommandContext(ctx, "systemctl", "start", s.Name).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("systemctl start %s: %s: %w", s.Name, strings.TrimSpace(string(out)), err)
		}
	} else {
		if out, err := exec.CommandContext(ctx, "systemctl", "stop", s.Name).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("systemctl stop %s: %s: %w", s.Name, strings.TrimSpace(string(out)), err)
		}
	}

	if s.Enable {
		if out, err := exec.CommandContext(ctx, "systemctl", "enable", s.Name).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("systemctl enable %s: %s: %w", s.Name, strings.TrimSpace(string(out)), err)
		}
	} else {
		if out, err := exec.CommandContext(ctx, "systemctl", "disable", s.Name).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("systemctl disable %s: %s: %w", s.Name, strings.TrimSpace(string(out)), err)
		}
	}

	msg := "started"
	if s.State == "stopped" {
		msg = "stopped"
	}
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: msg}, nil
}
