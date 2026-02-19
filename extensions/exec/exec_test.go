package exec

import (
	"context"
	"testing"
	"time"
)

func TestExec_Check(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		onlyIf   string
		wantSync bool
	}{
		{"no guard command", "", false},
		{"guard passes (true)", "true", true},
		{"guard fails (false)", "false", false},
		{"guard with args", "test -f /etc/os-release", true},
		{"guard file missing", "test -f /nonexistent/file", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Exec{Name: "test", Command: "echo", OnlyIf: tt.onlyIf}
			state, err := e.Check(ctx)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			if state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", state.InSync, tt.wantSync)
			}
		})
	}
}

func TestExec_Apply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		exec    *Exec
		wantErr bool
	}{
		{
			"simple echo",
			New("echo", "echo", "hello"),
			false,
		},
		{
			"command not found",
			New("bad", "/nonexistent/binary"),
			true,
		},
		{
			"false command fails",
			New("fail", "false"),
			true,
		},
		{
			"with working directory",
			&Exec{Name: "pwd", Command: "pwd", Dir: "/tmp"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.exec.Apply(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !result.Changed {
				t.Error("Changed should be true on success")
			}
		})
	}
}

func TestExec_ApplyRetries(t *testing.T) {
	ctx := context.Background()

	e := &Exec{
		Name:       "retry-test",
		Command:    "false",
		Retries:    3,
		RetryDelay: 10 * time.Millisecond,
	}

	start := time.Now()
	_, err := e.Apply(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from false command")
	}
	if elapsed < 20*time.Millisecond {
		t.Errorf("retries should have taken at least 20ms, took %v", elapsed)
	}
}

func TestExec_IDAndString(t *testing.T) {
	tests := []struct {
		name    string
		wantID  string
		wantStr string
	}{
		{"setup-fw", "exec:setup-fw", "Exec setup-fw"},
		{"install-deps", "exec:install-deps", "Exec install-deps"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.name, "echo")
			if e.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", e.ID(), tt.wantID)
			}
			if e.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", e.String(), tt.wantStr)
			}
		})
	}
}

func TestExec_IsCritical(t *testing.T) {
	e := New("test", "echo")
	if e.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	e.Critical = true
	if !e.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}
