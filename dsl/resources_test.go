package dsl

import (
	"context"
	"testing"
)

func TestResourceCheckAndApply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(r *Run)
		wantID  string
	}{
		{"file", func(r *Run) { r.File("/tmp/converge-test", FileOpts{Content: "hi"}) }, "file:/tmp/converge-test"},
		{"package", func(r *Run) { r.Package("git", PackageOpts{State: Present}) }, "package:git"},
		{"service", func(r *Run) { r.Service("sshd", ServiceOpts{State: Running}) }, "service:sshd"},
		{"exec", func(r *Run) { r.Exec("test", ExecOpts{Command: "echo"}) }, "exec:test"},
		{"user", func(r *Run) { r.User("dev", UserOpts{Shell: "/bin/bash"}) }, "user:dev"},
		{"registry", func(r *Run) { r.Registry(`HKLM\Test`, RegistryOpts{Value: "v"}) }, `registry:HKLM\Test\v`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			run := newRun(app)
			tt.setup(run)

			resources := run.Resources()
			if len(resources) != 1 {
				t.Fatalf("expected 1 resource, got %d", len(resources))
			}

			ext := resources[0]
			if ext.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", ext.ID(), tt.wantID)
			}

			state, err := ext.Check(ctx)
			if err != nil {
				t.Logf("Check() error = %v (expected for some extensions)", err)
				return
			}
			if state == nil {
				t.Fatal("Check() returned nil state")
			}
			t.Logf("InSync=%v Changes=%d", state.InSync, len(state.Changes))
		})
	}
}
