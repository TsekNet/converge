package dsl

import (
	"errors"
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
	errs     []error
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
		r.errs = append(r.errs, fmt.Errorf("%s: %v", ext.ID(), err))
		return
	}
	for _, dep := range deps {
		if err := r.graph.AddEdge(ext.ID(), dep); err != nil {
			r.errs = append(r.errs, fmt.Errorf("dependency %s -> %s: %v", ext.ID(), dep, err))
			return
		}
	}
}

// require validates that a required field is not empty, appends an error
// if it is, and returns false to signal the caller to bail out.
func (r *Run) require(resource, field, value string) bool {
	if value == "" {
		r.errs = append(r.errs, fmt.Errorf("%s requires %s (got empty string)", resource, field))
		return false
	}
	return true
}

// Err returns all errors encountered during blueprint execution, joined.
func (r *Run) Err() error {
	return errors.Join(r.errs...)
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
		r.errs = append(r.errs, fmt.Errorf("Include(%q): no app context", name))
		return
	}
	entry, ok := r.app.blueprints[name]
	if !ok {
		r.errs = append(r.errs, fmt.Errorf("Include(%q): blueprint not registered", name))
		return
	}
	entry.fn(r)
}

func (r *Run) File(path string, opts FileOpts) {
	if !r.require("File", "path", path) {
		return
	}
	r.addResource(newFileExtension(path, opts), opts.Meta.DependsOn)
}

func (r *Run) Package(name string, opts PackageOpts) {
	if !r.require("Package", "name", name) {
		return
	}
	if opts.State == "" {
		opts.State = Present
	}
	r.addResource(newPackageExtension(name, opts, r.platform.PkgManager), opts.Meta.DependsOn)
}

func (r *Run) Service(name string, opts ServiceOpts) {
	if !r.require("Service", "name", name) {
		return
	}
	if opts.State == "" {
		opts.State = Running
	}
	r.addResource(newServiceExtension(name, opts, r.platform.InitSystem), opts.Meta.DependsOn)
}

func (r *Run) Exec(name string, opts ExecOpts) {
	if !r.require("Exec", "name", name) {
		return
	}
	if !r.require("Exec", "command", opts.Command) {
		return
	}
	r.addResource(newExecExtension(name, opts), opts.Meta.DependsOn)
}

func (r *Run) User(name string, opts UserOpts) {
	if !r.require("User", "name", name) {
		return
	}
	r.addResource(newUserExtension(name, opts), opts.Meta.DependsOn)
}

func (r *Run) Firewall(name string, opts FirewallOpts) {
	if !r.require("Firewall", "name", name) {
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
	r.addResource(newFirewallExtension(name, opts), opts.Meta.DependsOn)
}
