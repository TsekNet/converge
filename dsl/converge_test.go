package dsl

import (
	"slices"
	"testing"
)

func TestApp_Blueprints(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
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
	t.Parallel()

	app := New()
	run := newRun(app)

	run.File("/etc/motd", FileOpts{Content: "hello"})
	run.Package("git", PackageOpts{State: Present})
	run.Service("sshd", ServiceOpts{State: Running})
	run.Exec("test", ExecOpts{Command: "echo hello"})
	run.User("dev", UserOpts{Shell: "/bin/bash"})
	run.Firewall("Allow SSH", FirewallOpts{Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow"})
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
		{5, "firewall:Allow SSH", "Firewall Allow SSH (tcp/22 allow)"},
	}

	resources := run.Resources()
	if len(resources) != len(tests) {
		t.Fatalf("Resources() count = %d, want %d", len(resources), len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.wantID, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()

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
			t.Parallel()
			if tt.value == "" {
				t.Errorf("Platform().%s should not be empty", tt.name)
			}
		})
	}
}

func TestRun_Include(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		registerBase  bool
		includeName   string
		wantResources int
		wantErr       bool
	}{
		{
			name:          "includes registered blueprint",
			registerBase:  true,
			includeName:   "base",
			wantResources: 2,
			wantErr:       false,
		},
		{
			name:          "error on missing blueprint",
			registerBase:  false,
			includeName:   "nonexistent",
			wantResources: 0,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := New()
			if tt.registerBase {
				app.Register("base", "base config", func(r *Run) {
					r.File("/etc/base", FileOpts{Content: "base"})
				})
			}

			run := newRun(app)
			run.Include(tt.includeName)

			if tt.registerBase {
				run.Package("vim", PackageOpts{State: Present})
			}

			if tt.wantErr {
				if run.Err() == nil {
					t.Error("Include() should set error on missing blueprint")
				}
				return
			}

			if got := len(run.Resources()); got != tt.wantResources {
				t.Errorf("resource count = %d, want %d", got, tt.wantResources)
			}
		})
	}
}

func TestRun_Firewall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fwName  string
		opts    FirewallOpts
		wantID  string
		wantStr string
		wantErr bool
	}{
		{
			name:    "defaults applied",
			fwName:  "DefaultsTest",
			opts:    FirewallOpts{Port: 443},
			wantID:  "firewall:DefaultsTest",
			wantStr: "Firewall DefaultsTest (tcp/443 allow)",
		},
		{
			name:   "absent state",
			fwName: "Remove SSH",
			opts:   FirewallOpts{Port: 22, State: Absent},
			wantID: "firewall:Remove SSH",
		},
		{
			name:    "error on empty name",
			fwName:  "",
			opts:    FirewallOpts{Port: 22},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			run := newRun(New())
			run.Firewall(tt.fwName, tt.opts)

			if tt.wantErr {
				if run.Err() == nil {
					t.Error("Firewall() should set error on empty name")
				}
				return
			}

			r := run.Resources()[0]
			if tt.wantID != "" && r.ID() != tt.wantID {
				t.Errorf("ID() = %q, want %q", r.ID(), tt.wantID)
			}
			if tt.wantStr != "" && r.String() != tt.wantStr {
				t.Errorf("String() = %q, want %q", r.String(), tt.wantStr)
			}
		})
	}
}
