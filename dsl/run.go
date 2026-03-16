package dsl

import (
	"fmt"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/platform"
)

// Run is the context passed to blueprints for declaring resources.
type Run struct {
	graph    *graph.Graph
	platform platform.Info
	app      *App
}

func newRun(app *App) *Run {
	return &Run{
		graph:    graph.New(),
		platform: platform.Detect(),
		app:      app,
	}
}

func (r *Run) addResource(ext extensions.Extension, deps []string) {
	if err := r.graph.AddNode(ext); err != nil {
		panic(fmt.Sprintf("converge: %v", err))
	}
	for _, dep := range deps {
		if err := r.graph.AddEdge(ext.ID(), dep); err != nil {
			panic(fmt.Sprintf("converge: dependency %s -> %s: %v", ext.ID(), dep, err))
		}
	}
}

// Graph returns the resource dependency graph for engine processing.
func (r *Run) Graph() *graph.Graph {
	return r.graph
}

// Resources returns collected extensions in insertion order.
func (r *Run) Resources() []extensions.Extension {
	return r.graph.OrderedExtensions()
}

// Platform returns detected OS, distro, package manager, and init system info.
func (r *Run) Platform() platform.Info {
	return r.platform
}

// Include runs another registered blueprint within this Run context.
func (r *Run) Include(name string) {
	if r.app == nil {
		panic(fmt.Sprintf("converge: Include(%q): no app context", name))
	}
	entry, ok := r.app.blueprints[name]
	if !ok {
		panic(fmt.Sprintf("converge: Include(%q): blueprint not registered", name))
	}
	entry.fn(r)
}

func (r *Run) File(path string, opts FileOpts) {
	mustNotBeEmpty("File", "path", path)
	r.addResource(newFileExtension(path, opts), opts.DependsOn)
}

func (r *Run) Package(name string, opts PackageOpts) {
	mustNotBeEmpty("Package", "name", name)
	if opts.State == "" {
		opts.State = Present
	}
	r.addResource(newPackageExtension(name, opts, r.platform.PkgManager), opts.DependsOn)
}

func (r *Run) Service(name string, opts ServiceOpts) {
	mustNotBeEmpty("Service", "name", name)
	if opts.State == "" {
		opts.State = Running
	}
	r.addResource(newServiceExtension(name, opts, r.platform.InitSystem), opts.DependsOn)
}

func (r *Run) Exec(name string, opts ExecOpts) {
	mustNotBeEmpty("Exec", "name", name)
	mustNotBeEmpty("Exec", "command", opts.Command)
	r.addResource(newExecExtension(name, opts), opts.DependsOn)
}

func (r *Run) User(name string, opts UserOpts) {
	mustNotBeEmpty("User", "name", name)
	r.addResource(newUserExtension(name, opts), opts.DependsOn)
}

func (r *Run) Firewall(name string, opts FirewallOpts) {
	mustNotBeEmpty("Firewall", "name", name)
	if opts.Protocol == "" {
		opts.Protocol = "tcp"
	}
	if opts.Direction == "" {
		opts.Direction = "inbound"
	}
	if opts.Action == "" {
		opts.Action = "allow"
	}
	r.addResource(newFirewallExtension(name, opts), opts.DependsOn)
}

func mustNotBeEmpty(resource, field, value string) {
	if value == "" {
		panic(fmt.Sprintf("converge: %s requires %s (got empty string)", resource, field))
	}
}
