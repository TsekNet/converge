package dsl

import (
	"fmt"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/platform"
)

// Run is the context passed to blueprints for declaring resources.
type Run struct {
	resources []extensions.Extension
	platform  platform.Info
	app       *App
}

func newRun(app *App) *Run {
	return &Run{
		platform: platform.Detect(),
		app:      app,
	}
}

func (r *Run) addResource(ext extensions.Extension) {
	r.resources = append(r.resources, ext)
}

// Resources returns the collected extensions for engine processing.
func (r *Run) Resources() []extensions.Extension {
	return r.resources
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
	r.addResource(newFileExtension(path, opts))
}

func (r *Run) Package(name string, opts PackageOpts) {
	mustNotBeEmpty("Package", "name", name)
	if opts.State == "" {
		opts.State = Present
	}
	r.addResource(newPackageExtension(name, opts, r.platform.PkgManager))
}

func (r *Run) Service(name string, opts ServiceOpts) {
	mustNotBeEmpty("Service", "name", name)
	if opts.State == "" {
		opts.State = Running
	}
	r.addResource(newServiceExtension(name, opts, r.platform.InitSystem))
}

func (r *Run) Exec(name string, opts ExecOpts) {
	mustNotBeEmpty("Exec", "name", name)
	mustNotBeEmpty("Exec", "command", opts.Command)
	r.addResource(newExecExtension(name, opts))
}

func (r *Run) User(name string, opts UserOpts) {
	mustNotBeEmpty("User", "name", name)
	r.addResource(newUserExtension(name, opts))
}

func (r *Run) Registry(key string, opts RegistryOpts) {
	mustNotBeEmpty("Registry", "key", key)
	r.addResource(newRegistryExtension(key, opts))
}

func mustNotBeEmpty(resource, field, value string) {
	if value == "" {
		panic(fmt.Sprintf("converge: %s requires %s (got empty string)", resource, field))
	}
}
