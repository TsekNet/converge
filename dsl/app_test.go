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

func TestApp_Version(t *testing.T) {
	app := New()
	if v := app.Version(); v == "" {
		t.Error("Version() should not be empty")
	}
}

func TestApp_RunPlan(t *testing.T) {
	tests := []struct {
		name     string
		bpName   string
		register string
		wantCode int
		wantErr  bool
	}{
		{"blueprint not found", "missing", "other", 11, true},
		{"valid blueprint", "test", "test", 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			app.Register(tt.register, "test blueprint", func(r *Run) {
				r.File("/etc/test", FileOpts{Content: "x"})
			})

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

func TestApp_RunPlan_DuplicateResources(t *testing.T) {
	app := New()
	app.Register("dupes", "duplicate test", func(r *Run) {
		r.File("/etc/motd", FileOpts{Content: "a"})
		r.File("/etc/motd", FileOpts{Content: "b"})
	})

	code, err := app.RunPlan("dupes", &testPrinter{})
	if err == nil {
		t.Fatal("expected error for duplicate resources")
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestApp_BuildGraph_NotFound(t *testing.T) {
	app := New()
	_, err := app.BuildGraph("missing")
	if err == nil {
		t.Fatal("expected error for missing blueprint")
	}
}

func TestApp_BuildGraph_AutoEdges(t *testing.T) {
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
