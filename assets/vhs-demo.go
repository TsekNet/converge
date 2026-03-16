//go:build ignore

// vhs-demo produces representative converge output for the VHS demo recording.
// Supports: converge plan baseline, converge serve baseline --timeout 1s
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/output"
)

type stubExt struct {
	id   string
	name string
}

func (s *stubExt) ID() string                                          { return s.id }
func (s *stubExt) String() string                                      { return s.name }
func (s *stubExt) Check(_ context.Context) (*extensions.State, error)  { return nil, nil }
func (s *stubExt) Apply(_ context.Context) (*extensions.Result, error) { return nil, nil }

var _ extensions.Extension = (*stubExt)(nil)

type resource struct {
	ext    extensions.Extension
	state  *extensions.State
	result *extensions.Result
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: converge <plan|serve> <blueprint> [flags]")
		os.Exit(1)
	}

	cmd := args[0]
	blueprint := "baseline"
	if len(args) >= 2 {
		blueprint = args[1]
	}

	p := output.NewTerminalPrinter()
	p.SetMaxNameLen(28)
	p.Banner("dev")
	p.BlueprintHeader(blueprint)

	resources := baselineResources()

	switch cmd {
	case "plan":
		runPlan(p, resources)
	case "serve":
		runApply(p, resources)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func runPlan(p *output.TerminalPrinter, resources []resource) {
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

func runApply(p *output.TerminalPrinter, resources []resource) {
	changed, ok := 0, 0
	for i, r := range resources {
		p.ApplyStart(r.ext, i+1, len(resources))
		// Simulate work: longer for packages, shorter for files
		delay := 120 * time.Millisecond
		if r.ext.ID() == "package:neovim" {
			delay = 800 * time.Millisecond
		}
		time.Sleep(delay)
		p.ApplyResult(r.ext, r.result)
		if r.result.Changed {
			changed++
		} else {
			ok++
		}
	}
	total := changed + ok
	p.Summary(changed, ok, 0, total, 2400)
}

func baselineResources() []resource {
	return []resource{
		{
			ext: &stubExt{"file:/etc/motd", "File /etc/motd"},
			state: &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "content", From: "(absent)", To: "Managed by Converge", Action: "add"},
					{Property: "mode", To: "0644", Action: "add"},
				},
			},
			result: &extensions.Result{
				Changed: true, Status: extensions.StatusChanged, Message: "created",
				Duration: 12 * time.Millisecond,
				Changes: []extensions.Change{
					{Property: "content", From: "(absent)", To: "Managed by Converge", Action: "add"},
					{Property: "mode", To: "0644", Action: "add"},
				},
			},
		},
		{
			ext:    &stubExt{"package:git", "Package git"},
			state:  &extensions.State{InSync: true},
			result: &extensions.Result{Status: extensions.StatusOK, Duration: 340 * time.Millisecond},
		},
		{
			ext:    &stubExt{"package:curl", "Package curl"},
			state:  &extensions.State{InSync: true},
			result: &extensions.Result{Status: extensions.StatusOK, Duration: 280 * time.Millisecond},
		},
		{
			ext: &stubExt{"package:neovim", "Package neovim"},
			state: &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "neovim", To: "install via apt", Action: "add"},
				},
			},
			result: &extensions.Result{
				Changed: true, Status: extensions.StatusChanged, Message: "installed",
				Duration: 1800 * time.Millisecond,
				Changes: []extensions.Change{
					{Property: "neovim", To: "install via apt", Action: "add"},
				},
			},
		},
		{
			ext:    &stubExt{"firewall:Allow SSH", "Firewall Allow SSH (tcp/22 allow)"},
			state:  &extensions.State{InSync: true},
			result: &extensions.Result{Status: extensions.StatusOK, Duration: 1 * time.Millisecond},
		},
		{
			ext: &stubExt{"service:sshd", "Service sshd"},
			state: &extensions.State{
				InSync: false,
				Changes: []extensions.Change{
					{Property: "state", From: "stopped", To: "running", Action: "modify"},
					{Property: "enable", To: "true", Action: "add"},
				},
			},
			result: &extensions.Result{
				Changed: true, Status: extensions.StatusChanged, Message: "started",
				Duration: 420 * time.Millisecond,
				Changes: []extensions.Change{
					{Property: "state", From: "stopped", To: "running", Action: "modify"},
					{Property: "enable", To: "true", Action: "add"},
				},
			},
		},
	}
}
