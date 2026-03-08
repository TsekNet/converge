package dsl

import (
	"slices"
	"testing"
)

func TestResourceState(t *testing.T) {
	tests := []struct {
		name  string
		state ResourceState
		want  string
	}{
		{"present", Present, "present"},
		{"absent", Absent, "absent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.state); got != tt.want {
				t.Errorf("ResourceState = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceState(t *testing.T) {
	tests := []struct {
		name  string
		state ServiceState
		want  string
	}{
		{"running", Running, "running"},
		{"stopped", Stopped, "stopped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.state); got != tt.want {
				t.Errorf("ServiceState = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApp_Blueprints(t *testing.T) {
	tests := []struct {
		name       string
		registered []string
		want       []string
	}{
		{"empty", nil, nil},
		{"single", []string{"web"}, []string{"web"}},
		{"multiple sorted", []string{"zebra", "alpha", "middle"}, []string{"alpha", "middle", "zebra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			for _, name := range tt.registered {
				app.Register(name, "", func(r *Run) {})
			}
			got := app.Blueprints()
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			var names []string
			for _, item := range got {
				names = append(names, item.Name)
			}
			if !slices.Equal(names, tt.want) {
				t.Errorf("Blueprints() names = %v, want %v", names, tt.want)
			}
		})
	}
}

func TestRun_ResourceIDs(t *testing.T) {
	app := New()
	run := newRun(app)

	run.File("/etc/motd", FileOpts{Content: "hello"})
	run.Package("git", PackageOpts{State: Present})
	run.Service("sshd", ServiceOpts{State: Running})
	run.Exec("test", ExecOpts{Command: "echo hello"})
	run.User("dev", UserOpts{Shell: "/bin/bash"})
	tests := []struct {
		index   int
		wantID  string
		wantStr string
	}{
		{0, "file:/etc/motd", "File /etc/motd"},
		{1, "package:git", "Package git"},
		{2, "service:sshd", "Service sshd"},
		{3, "exec:test", "Exec test"},
		{4, "user:dev", "User dev"},
	}

	resources := run.Resources()
	if len(resources) != len(tests) {
		t.Fatalf("Resources() count = %d, want %d", len(resources), len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.wantID, func(t *testing.T) {
			r := resources[tt.index]
			if r.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", r.ID(), tt.wantID)
			}
			if r.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", r.String(), tt.wantStr)
			}
		})
	}
}

func TestRun_Platform(t *testing.T) {
	run := newRun(New())
	p := run.Platform()

	tests := []struct {
		name  string
		value string
	}{
		{"OS", p.OS},
		{"Arch", p.Arch},
		{"Distro", p.Distro},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("Platform().%s should not be empty", tt.name)
			}
		})
	}
}

func TestRun_Include(t *testing.T) {
	app := New()
	app.Register("base", "base config", func(r *Run) {
		r.File("/etc/base", FileOpts{Content: "base"})
	})

	run := newRun(app)
	run.Include("base")
	run.Package("vim", PackageOpts{State: Present})

	if got := len(run.Resources()); got != 2 {
		t.Errorf("resource count = %d, want 2", got)
	}
}

func TestRun_Include_PanicsOnMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Include() should panic on missing blueprint")
		}
	}()

	app := New()
	run := newRun(app)
	run.Include("nonexistent")
}
