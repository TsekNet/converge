package dsl

import (
	"testing"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/output"
)

type testPrinter struct{}

func (p *testPrinter) Banner(_ string)                                          {}
func (p *testPrinter) BlueprintHeader(_ string)                                 {}
func (p *testPrinter) ResourceChecking(_ extensions.Extension, _, _ int)        {}
func (p *testPrinter) PlanResult(_ extensions.Extension, _ *extensions.State)   {}
func (p *testPrinter) ApplyStart(_ extensions.Extension, _, _ int)              {}
func (p *testPrinter) ApplyResult(_ extensions.Extension, _ *extensions.Result) {}
func (p *testPrinter) Summary(_, _, _, _ int, _ int64)                          {}
func (p *testPrinter) PlanSummary(_, _, _ int)                                  {}
func (p *testPrinter) Error(_ extensions.Extension, _ error)                    {}

var _ output.Printer = (*testPrinter)(nil)

func TestApp_RunPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bpName   string
		register string
		setupFn  func(r *Run)
		wantCode int
		wantErr  bool
	}{
		{
			name:     "blueprint not found",
			bpName:   "missing",
			register: "other",
			setupFn:  func(r *Run) { r.File("/etc/test", FileOpts{Content: "x"}) },
			wantCode: 11,
			wantErr:  true,
		},
		{
			name:     "valid blueprint",
			bpName:   "test",
			register: "test",
			setupFn:  func(r *Run) { r.File("/etc/test", FileOpts{Content: "x"}) },
			wantCode: 5,
			wantErr:  false,
		},
		{
			name:     "duplicate resources",
			bpName:   "dupes",
			register: "dupes",
			setupFn: func(r *Run) {
				r.File("/etc/motd", FileOpts{Content: "a"})
				r.File("/etc/motd", FileOpts{Content: "b"})
			},
			wantCode: 1,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := New()
			app.Register(tt.register, "test blueprint", tt.setupFn)

			code, err := app.RunPlan(tt.bpName, &testPrinter{})
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestApp_BuildGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		blueprint string
		register  bool
		setupFn   func(r *Run)
		wantErr   bool
	}{
		{
			name:      "missing blueprint",
			blueprint: "missing",
			register:  false,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := New()
			if tt.register {
				app.Register(tt.blueprint, "test", tt.setupFn)
			}
			_, err := app.BuildGraph(tt.blueprint)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApp_BuildGraph_AutoEdges(t *testing.T) {
	t.Parallel()

	app := New()
	app.Register("test", "auto-edge test", func(r *Run) {
		r.Package("nginx", PackageOpts{State: Present})
		r.Service("nginx", ServiceOpts{State: Running})
	})

	g, err := app.BuildGraph("test")
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	// Auto-edge should create service:nginx -> package:nginx.
	layers, _ := g.TopologicalLayers()
	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2 (package then service)", len(layers))
	}
}
