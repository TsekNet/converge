//go:build windows

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Check connects to the Windows Service Control Manager and compares current state/startup type.
func (s *Service) Check(_ context.Context) (*extensions.State, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connect to SCM: %w", err)
	}
	defer m.Disconnect()

	handle, err := m.OpenService(s.Name)
	if err != nil {
		return nil, fmt.Errorf("open service %s: %w", s.Name, err)
	}
	defer handle.Close()

	var changes []extensions.Change

	status, err := handle.Query()
	if err != nil {
		return nil, fmt.Errorf("query service %s: %w", s.Name, err)
	}

	isRunning := status.State == svc.Running
	wantRunning := s.State == "running"
	if isRunning != wantRunning {
		from, to := "running", "stopped"
		if wantRunning {
			from, to = "stopped", "running"
		}
		changes = append(changes, extensions.Change{Property: "state", From: from, To: to, Action: "modify"})
	}

	if s.StartupType != "" {
		cfg, err := handle.Config()
		if err != nil {
			return nil, fmt.Errorf("config service %s: %w", s.Name, err)
		}
		currentType := startTypeToString(cfg.StartType, cfg.DelayedAutoStart)
		desired := strings.ToLower(s.StartupType)
		if currentType != desired {
			changes = append(changes, extensions.Change{Property: "startup_type", From: currentType, To: desired, Action: "modify"})
		}
	}

	return &extensions.State{InSync: len(changes) == 0, Changes: changes}, nil
}

// Apply updates startup type via UpdateConfig and starts/stops the service as needed.
func (s *Service) Apply(_ context.Context) (*extensions.Result, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connect to SCM: %w", err)
	}
	defer m.Disconnect()

	handle, err := m.OpenService(s.Name)
	if err != nil {
		return nil, fmt.Errorf("open service %s: %w", s.Name, err)
	}
	defer handle.Close()

	if s.StartupType != "" {
		cfg, err := handle.Config()
		if err != nil {
			return nil, fmt.Errorf("config service %s: %w", s.Name, err)
		}
		desired := strings.ToLower(s.StartupType)
		newStart, delayed := parseStartupType(desired)
		cfg.StartType = newStart
		cfg.DelayedAutoStart = delayed
		if err := handle.UpdateConfig(cfg); err != nil {
			return nil, fmt.Errorf("update config %s: %w", s.Name, err)
		}
	}

	if s.State == "stopped" {
		status, err := handle.Query()
		if err != nil {
			return nil, fmt.Errorf("query service %s: %w", s.Name, err)
		}
		if status.State == svc.Running {
			if _, err := handle.Control(svc.Stop); err != nil {
				return nil, fmt.Errorf("stop service %s: %w", s.Name, err)
			}
			if err := waitForState(handle, svc.Stopped, 30*time.Second); err != nil {
				return nil, fmt.Errorf("wait stop %s: %w", s.Name, err)
			}
		}
	} else {
		status, err := handle.Query()
		if err != nil {
			return nil, fmt.Errorf("query service %s: %w", s.Name, err)
		}
		if status.State != svc.Running {
			if err := handle.Start(); err != nil {
				return nil, fmt.Errorf("start service %s: %w", s.Name, err)
			}
		}
	}

	msg := "started"
	if s.State == "stopped" {
		msg = "stopped"
	}
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: msg}, nil
}

// startTypeToString maps SCM start type constants to human-readable names.
func startTypeToString(st uint32, delayed bool) string {
	switch st {
	case mgr.StartAutomatic:
		if delayed {
			return "delayed-auto"
		}
		return "auto"
	case mgr.StartManual:
		return "manual"
	case mgr.StartDisabled:
		return "disabled"
	default:
		return fmt.Sprintf("unknown(%d)", st)
	}
}

func parseStartupType(s string) (uint32, bool) {
	switch s {
	case "auto":
		return mgr.StartAutomatic, false
	case "delayed-auto":
		return mgr.StartAutomatic, true
	case "manual":
		return mgr.StartManual, false
	case "disabled":
		return mgr.StartDisabled, false
	default:
		return mgr.StartManual, false
	}
}

// waitForState polls the service until it reaches the desired state or times out.
func waitForState(handle *mgr.Service, desired svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := handle.Query()
		if err != nil {
			return err
		}
		if status.State == desired {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for state %d", desired)
}
