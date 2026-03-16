package dsl

import (
	"fmt"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/platform"
)

// Run is the context passed to blueprints for declaring resources.
// Errors during resource declaration are accumulated, not panicked.
type Run struct {
	graph    *graph.Graph
	platform platform.Info
	app      *App
	err      error // first error encountered during blueprint execution
}

func newRun(app *App) *Run {
	return &Run{
		graph:    graph.New(),
		platform: platform.Detect(),
		app:      app,
	}
}

func (r *Run) addResource(ext extensions.Extension, deps []string) {
	if r.err != nil {
		return // stop processing after first error
	}
	if err := r.graph.AddNode(ext); err != nil {
		r.err = fmt.Errorf("%s: %v", ext.ID(), err)
		return
	}
	for _, dep := range deps {
		if err := r.graph.AddEdge(ext.ID(), dep); err != nil {
			r.err = fmt.Errorf("dependency %s -> %s: %v", ext.ID(), dep, err)
			return
		}
	}
}

// Err returns the first error encountered during blueprint execution.
func (r *Run) Err() error {
	return r.err
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
	if r.err != nil {
		return
	}
	if r.app == nil {
		r.err = fmt.Errorf("Include(%q): no app context", name)
		return
	}
	entry, ok := r.app.blueprints[name]
	if !ok {
		r.err = fmt.Errorf("Include(%q): blueprint not registered", name)
		return
	}
	entry.fn(r)
}

func (r *Run) File(path string, opts FileOpts) {
	if err := requireNotEmpty("File", "path", path); err != nil {
		r.err = err
		return
	}
	r.addResource(newFileExtension(path, opts), opts.DependsOn)
}

func (r *Run) Package(name string, opts PackageOpts) {
	if err := requireNotEmpty("Package", "name", name); err != nil {
		r.err = err
		return
	}
	if opts.State == "" {
		opts.State = Present
	}
	r.addResource(newPackageExtension(name, opts, r.platform.PkgManager), opts.DependsOn)
}

func (r *Run) Service(name string, opts ServiceOpts) {
	if err := requireNotEmpty("Service", "name", name); err != nil {
		r.err = err
		return
	}
	if opts.State == "" {
		opts.State = Running
	}
	r.addResource(newServiceExtension(name, opts, r.platform.InitSystem), opts.DependsOn)
}

func (r *Run) Exec(name string, opts ExecOpts) {
	if err := requireNotEmpty("Exec", "name", name); err != nil {
		r.err = err
		return
	}
	if err := requireNotEmpty("Exec", "command", opts.Command); err != nil {
		r.err = err
		return
	}
	r.addResource(newExecExtension(name, opts), opts.DependsOn)
}

func (r *Run) User(name string, opts UserOpts) {
	if err := requireNotEmpty("User", "name", name); err != nil {
		r.err = err
		return
	}
	r.addResource(newUserExtension(name, opts), opts.DependsOn)
}

func (r *Run) Firewall(name string, opts FirewallOpts) {
	if err := requireNotEmpty("Firewall", "name", name); err != nil {
		r.err = err
		return
	}
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

func requireNotEmpty(resource, field, value string) error {
	if value == "" {
		return fmt.Errorf("%s requires %s (got empty string)", resource, field)
	}
	return nil
}
