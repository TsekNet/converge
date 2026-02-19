package dsl

import (
	"fmt"
	"maps"
	"os"
	"runtime"
	"slices"

	"github.com/TsekNet/converge/internal/engine"
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

// Extensions returns the built-in resource types.
func (a *App) Extensions() []Item {
	return []Item{
		{"File", "Manage file content, permissions, and ownership"},
		{"Package", "Install and remove system packages"},
		{"Service", "Manage system services"},
		{"Exec", "Run commands with guards and retries"},
		{"User", "Manage local user accounts"},
		{"Registry", "Manage Windows registry keys"},
	}
}

func (a *App) Version() string {
	return version.Version
}

func (a *App) RunPlan(name string, printer output.Printer) (int, error) {
	entry, ok := a.blueprints[name]
	if !ok {
		return 11, fmt.Errorf("blueprint %q not found", name)
	}

	run := newRun(a)
	entry.fn(run)

	resources := run.Resources()
	if err := engine.CheckDuplicates(resources); err != nil {
		return 1, err
	}

	return engine.RunPlan(resources, printer, a.EngineOpts)
}

func (a *App) RunApply(name string, printer output.Printer) (int, error) {
	entry, ok := a.blueprints[name]
	if !ok {
		return 11, fmt.Errorf("blueprint %q not found", name)
	}

	if !isRoot() {
		return 10, fmt.Errorf("converge apply requires root/administrator privileges")
	}

	run := newRun(a)
	entry.fn(run)

	resources := run.Resources()
	if err := engine.CheckDuplicates(resources); err != nil {
		return 1, err
	}

	return engine.RunApply(resources, printer, a.EngineOpts)
}

func isRoot() bool {
	if runtime.GOOS == "windows" {
		f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		if err != nil {
			return false
		}
		f.Close()
		return true
	}
	return os.Geteuid() == 0
}
