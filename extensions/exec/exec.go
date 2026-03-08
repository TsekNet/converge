package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
)

// Exec runs an arbitrary command. Use OnlyIf as a guard: if the guard exits 0, the resource is already in sync.
type Exec struct {
	Name       string
	Command    string
	Args       []string
	OnlyIf     string // if exit 0, skip Apply (guard command)
	Dir        string
	Env        []string
	Retries    int
	RetryDelay time.Duration
	Critical   bool
}

func New(name, command string, args ...string) *Exec {
	return &Exec{Name: name, Command: command, Args: args}
}

func (e *Exec) ID() string       { return fmt.Sprintf("exec:%s", e.Name) }
func (e *Exec) String() string   { return fmt.Sprintf("Exec %s", e.Name) }
func (e *Exec) IsCritical() bool { return e.Critical }

// checkGuard runs the OnlyIf command. Exit 0 = in sync (skip Apply), non-zero = needs Apply.
func (e *Exec) checkGuard(ctx context.Context) (*extensions.State, error) {
	if e.OnlyIf == "" {
		return &extensions.State{InSync: false}, nil
	}

	parts := strings.Fields(e.OnlyIf)
	if len(parts) == 0 {
		return &extensions.State{InSync: false}, nil
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if e.Dir != "" {
		cmd.Dir = e.Dir
	}
	if len(e.Env) > 0 {
		cmd.Env = append(os.Environ(), e.Env...)
	}

	err := cmd.Run()
	if err == nil {
		return &extensions.State{InSync: true}, nil
	}

	return &extensions.State{
		InSync: false,
		Changes: []extensions.Change{
			{Property: "command", To: e.Command, Action: "add"},
		},
	}, nil
}

func (e *Exec) Check(ctx context.Context) (*extensions.State, error) {
	return e.checkGuard(ctx)
}

func (e *Exec) Apply(ctx context.Context) (*extensions.Result, error) {
	retries := e.Retries
	if retries <= 0 {
		retries = 1
	}
	delay := e.RetryDelay
	if delay <= 0 {
		delay = time.Second
	}

	var lastErr error
	for attempt := range retries {
		cmd := exec.CommandContext(ctx, e.Command, e.Args...)
		if e.Dir != "" {
			cmd.Dir = e.Dir
		}
		if len(e.Env) > 0 {
			cmd.Env = append(os.Environ(), e.Env...)
		}

		output, err := cmd.CombinedOutput()
		if err == nil {
			return &extensions.Result{
				Changed: true,
				Status:  extensions.StatusChanged,
				Message: "executed",
			}, nil
		}

		lastErr = fmt.Errorf("%s (attempt %d/%d): %s: %w", e.Command, attempt+1, retries, strings.TrimSpace(string(output)), err)
		if attempt < retries-1 {
			time.Sleep(delay)
		}
	}

	return nil, lastErr
}
