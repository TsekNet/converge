package dsl

import (
	"fmt"
	"maps"
	"slices"

	"github.com/TsekNet/converge/internal/engine"
	"github.com/TsekNet/converge/internal/exit"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/graph/autoedge"
	"github.com/TsekNet/converge/internal/output"
	"github.com/TsekNet/converge/internal/version"
)

// App is the top-level Converge application.
type App struct {
	blueprints map[string]blueprintEntry
	EngineOpts engine.Options
}

type blueprintEntry struct {
	fn   Blueprint
	desc string
}

// Item is a name-description pair used for listing blueprints and extensions.
type Item struct {
	Name        string
	Description string
}

func New() *App {
	return &App{
		blueprints: make(map[string]blueprintEntry),
		EngineOpts: engine.DefaultOptions(),
	}
}

func (a *App) Register(name, description string, bp Blueprint) {
	a.blueprints[name] = blueprintEntry{fn: bp, desc: description}
}

// Blueprints returns registered blueprints sorted by name.
func (a *App) Blueprints() []Item {
	names := slices.Sorted(maps.Keys(a.blueprints))
	out := make([]Item, len(names))
	for i, n := range names {
		out[i] = Item{Name: n, Description: a.blueprints[n].desc}
	}
	return out
}

// Extensions returns built-in capabilities (resources and helpers).
func (a *App) Extensions() []Item {
	return []Item{
		{"File", "Manage file content, permissions, and ownership"},
		{"Package", "Install and remove system packages"},
		{"Service", "Manage system services"},
		{"Exec", "Run commands with guards and retries"},
		{"User", "Manage local user accounts"},
		{"Firewall", "Manage host firewall rules (nftables, pf, Windows Firewall)"},
		{"Registry", "Manage Windows registry keys (Windows only)"},
		{"SecurityPolicy", "Manage Windows password and lockout policies (Windows only)"},
		{"AuditPolicy", "Manage Windows advanced audit policy (Windows only)"},
		{"Sysctl", "Manage Linux kernel parameters via /proc/sys (Linux only)"},
		{"Plist", "Manage macOS preference domains (macOS only)"},
		{"InShard", "Percentage-based rollout sharding by hardware serial (helper)"},
		{"Secret", "Retrieve and decrypt config values with AES-256-GCM (helper)"},
	}
}

func (a *App) Version() string {
	return version.Version
}

// BuildGraph constructs the resource dependency graph for a named blueprint.
func (a *App) BuildGraph(name string) (*graph.Graph, error) {
	entry, ok := a.blueprints[name]
	if !ok {
		return nil, fmt.Errorf("blueprint %q not found", name)
	}

	run := newRun(a)
	entry.fn(run)
	if run.Err() != nil {
		return nil, run.Err()
	}

	if err := autoedge.AddAutoEdges(run.Graph()); err != nil {
		return nil, fmt.Errorf("auto-edges: %w", err)
	}

	return run.Graph(), nil
}

func (a *App) RunPlan(name string, printer output.Printer) (int, error) {
	if _, ok := a.blueprints[name]; !ok {
		return exit.NotFound, fmt.Errorf("blueprint %q not found", name)
	}
	g, err := a.BuildGraph(name)
	if err != nil {
		return exit.Error, err
	}
	return engine.RunPlanDAG(g, printer, a.EngineOpts)
}

