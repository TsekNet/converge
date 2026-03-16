package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/output"
)

type mockExtension struct {
	id       string
	name     string
	inSync   bool
	applyErr error
	checkErr error
}

func (m *mockExtension) ID() string     { return m.id }
func (m *mockExtension) String() string { return m.name }

func (m *mockExtension) Check(_ context.Context) (*extensions.State, error) {
	if m.checkErr != nil {
		return nil, m.checkErr
	}
	return &extensions.State{InSync: m.inSync}, nil
}

func (m *mockExtension) Apply(_ context.Context) (*extensions.Result, error) {
	if m.applyErr != nil {
		return nil, m.applyErr
	}
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "applied"}, nil
}

type criticalMock struct {
	mockExtension
	critical bool
}

func (c *criticalMock) IsCritical() bool { return c.critical }

type discardPrinter struct{}

func (d *discardPrinter) Banner(_ string)                                          {}
func (d *discardPrinter) BlueprintHeader(_ string)                                 {}
func (d *discardPrinter) ResourceChecking(_ extensions.Extension, _, _ int)        {}
func (d *discardPrinter) PlanResult(_ extensions.Extension, _ *extensions.State)   {}
func (d *discardPrinter) ApplyStart(_ extensions.Extension, _, _ int)              {}
func (d *discardPrinter) ApplyResult(_ extensions.Extension, _ *extensions.Result) {}
func (d *discardPrinter) Summary(_, _, _, _ int, _ int64)                          {}
func (d *discardPrinter) PlanSummary(_, _, _ int)                                  {}
func (d *discardPrinter) Error(_ extensions.Extension, _ error)                    {}

var _ output.Printer = (*discardPrinter)(nil)

func makeGraph(exts ...extensions.Extension) *graph.Graph {
	g := graph.New()
	for _, e := range exts {
		g.AddNode(e)
	}
	return g
}

func TestRunPlanDAG(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exts     []extensions.Extension
		wantCode int
		wantErr  bool
	}{
		{"all converged", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", inSync: true},
		}, 0, false},
		{"pending changes", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", inSync: false},
		}, 5, false},
		{"check error", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", checkErr: fmt.Errorf("denied")},
		}, 1, true},
		{"empty", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := makeGraph(tt.exts...)
			code, err := RunPlanDAG(g, &discardPrinter{}, DefaultOptions())
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestRunApplyDAG(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exts     []extensions.Extension
		wantCode int
	}{
		{"all converged", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", inSync: true},
		}, 0},
		{"changes applied", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", inSync: false},
		}, 2},
		{"partial failure", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", inSync: false},
			&mockExtension{id: "pkg:git", name: "Package git", inSync: false, applyErr: fmt.Errorf("fail")},
		}, 3},
		{"all failed", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", inSync: false, applyErr: fmt.Errorf("fail")},
		}, 4},
		{"check error", []extensions.Extension{
			&mockExtension{id: "file:/a", name: "File /a", checkErr: fmt.Errorf("denied")},
		}, 4},
		{"empty", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := makeGraph(tt.exts...)
			code, err := RunApplyDAG(g, &discardPrinter{}, DefaultOptions())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestRunApplyDAG_Parallel(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()
	opts.Parallel = 2

	g := makeGraph(
		&mockExtension{id: "file:/a", name: "File /a", inSync: false},
		&mockExtension{id: "file:/b", name: "File /b", inSync: true},
		&mockExtension{id: "pkg:git", name: "Package git", inSync: false},
	)
	code, err := RunApplyDAG(g, &discardPrinter{}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRunApplyDAG_CriticalFailure(t *testing.T) {
	t.Parallel()

	g := makeGraph(
		&criticalMock{
			mockExtension: mockExtension{id: "file:/a", name: "File /a", inSync: false, applyErr: fmt.Errorf("fail")},
			critical:      true,
		},
		&mockExtension{id: "file:/b", name: "File /b", inSync: false},
	)
	code, err := RunApplyDAG(g, &discardPrinter{}, DefaultOptions())
	if code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}
	if err == nil {
		t.Error("expected error for critical failure")
	}
}

func TestRunApplyDAG_WithDependencies(t *testing.T) {
	t.Parallel()

	g := graph.New()
	pkg := &mockExtension{id: "package:nginx", name: "Package nginx", inSync: false}
	svc := &mockExtension{id: "service:nginx", name: "Service nginx", inSync: false}
	g.AddNode(pkg)
	g.AddNode(svc)
	g.AddEdge("service:nginx", "package:nginx")

	code, err := RunApplyDAG(g, &discardPrinter{}, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (changed)", code)
	}
}
