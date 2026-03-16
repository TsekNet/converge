//go:build ignore

// vhs-demo produces representative converge plan output for the VHS demo recording.
// Uses the output package with mock extensions. Run via: go run ./assets/vhs-demo.go baseline
package main

import (
	"context"
	"flag"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/output"
)

type stubExt struct {
	id   string
	name string
}

func (s *stubExt) ID() string                                    { return s.id }
func (s *stubExt) String() string                                { return s.name }
func (s *stubExt) Check(_ context.Context) (*extensions.State, error) { return nil, nil }
func (s *stubExt) Apply(_ context.Context) (*extensions.Result, error) { return nil, nil }

var _ extensions.Extension = (*stubExt)(nil)

func main() {
	flag.Parse()
	blueprint := "baseline"
	// Accept "converge plan <blueprint>" style args
	args := flag.Args()
	if len(args) >= 2 && args[0] == "plan" {
		blueprint = args[1]
	} else if len(args) >= 1 && args[0] != "plan" {
		blueprint = args[0]
	}

	p := output.NewTerminalPrinter()
	p.SetMaxNameLen(28)

	// Banner + header
	p.Banner("0.0.5")
	p.BlueprintHeader(blueprint)

	type resource struct {
		ext   extensions.Extension
		state *extensions.State
	}

	var resources []resource
	switch blueprint {
	case "server":
		resources = []resource{
			{&stubExt{"file:/etc/motd", "File /etc/motd"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "content", From: "(absent)", To: "Managed by Converge", Action: "add"},
				},
			}},
			{&stubExt{"package:nginx", "Package nginx"}, &extensions.State{InSync: true}},
			{&stubExt{"service:nginx", "Service nginx"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "state", From: "stopped", To: "running", Action: "modify"},
					{Property: "enable", To: "true", Action: "add"},
				},
			}},
			{&stubExt{"firewall:Allow HTTP", "Firewall Allow HTTP"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "rule", From: "absent", To: "present", Action: "add"},
				},
			}},
			{&stubExt{"firewall:Allow HTTPS", "Firewall Allow HTTPS"}, &extensions.State{InSync: true}},
			{&stubExt{"firewall:Block SMTP out", "Firewall Block SMTP out"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "rule", From: "absent", To: "present", Action: "add"},
				},
			}},
		}
	default:
		resources = []resource{
			{&stubExt{"file:/etc/motd", "File /etc/motd"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "content", From: "(absent)", To: "Managed by Converge", Action: "add"},
					{Property: "mode", To: "0644", Action: "add"},
				},
			}},
			{&stubExt{"package:git", "Package git"}, &extensions.State{InSync: true}},
			{&stubExt{"package:curl", "Package curl"}, &extensions.State{InSync: true}},
			{&stubExt{"package:neovim", "Package neovim"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "state", From: "absent", To: "present", Action: "add"},
				},
			}},
			{&stubExt{"firewall:Allow SSH", "Firewall Allow SSH"}, &extensions.State{InSync: true}},
			{&stubExt{"service:sshd", "Service sshd"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "enable", To: "true", Action: "add"},
					{Property: "state", From: "stopped", To: "running", Action: "modify"},
				},
			}},
			{&stubExt{"user:devuser", "User devuser"}, &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "groups", To: "[sudo]", Action: "add"},
					{Property: "shell", To: "/bin/bash", Action: "add"},
				},
			}},
		}
	}

	for i, r := range resources {
		p.ResourceChecking(r.ext, i+1, len(resources))
		time.Sleep(80 * time.Millisecond)
		p.PlanResult(r.ext, r.state)
	}

	pending := 0
	for _, r := range resources {
		if !r.state.InSync {
			pending++
		}
	}
	p.PlanSummary(pending, len(resources)-pending, len(resources))
}
