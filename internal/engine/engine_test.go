package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/TsekNet/converge/extensions"
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

func TestCheckDuplicates(t *testing.T) {
	tests := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{"no duplicates", []string{"file:/a", "package:git"}, false},
		{"empty list", nil, false},
		{"single resource", []string{"file:/a"}, false},
		{"duplicate IDs", []string{"file:/a", "file:/a"}, true},
		{"duplicate among many", []string{"file:/a", "package:git", "file:/a"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources := make([]extensions.Extension, len(tt.ids))
			for i, id := range tt.ids {
				resources[i] = &mockExtension{id: id, name: id}
			}
			err := CheckDuplicates(resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckDuplicates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 5*time.Minute)
	}
	if opts.Parallel != 1 {
		t.Errorf("Parallel = %d, want 1", opts.Parallel)
	}
}

func TestIsCritical(t *testing.T) {
	if isCritical(&mockExtension{id: "a", name: "a"}) {
		t.Error("regular extension should not be critical")
	}
	if !isCritical(&criticalMock{mockExtension: mockExtension{id: "b", name: "b"}, critical: true}) {
		t.Error("critical extension should be critical")
	}
	if isCritical(&criticalMock{mockExtension: mockExtension{id: "c", name: "c"}, critical: false}) {
		t.Error("non-critical extension should not be critical")
	}
}

func TestWithTimeout(t *testing.T) {
	ctx := context.Background()

	rctx, cancel := withTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, ok := rctx.Deadline(); !ok {
		t.Error("expected deadline when timeout > 0")
	}

	rctx2, cancel2 := withTimeout(ctx, 0)
	defer cancel2()
	if _, ok := rctx2.Deadline(); ok {
		t.Error("should not have deadline when timeout is 0")
	}
}

func TestRunPlan(t *testing.T) {
	tests := []struct {
		name      string
		resources []extensions.Extension
		wantCode  int
		wantErr   bool
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
			code, err := RunPlan(tt.resources, &discardPrinter{}, DefaultOptions())
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestRunApply(t *testing.T) {
	tests := []struct {
		name      string
		resources []extensions.Extension
		wantCode  int
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
			code, err := RunApply(tt.resources, &discardPrinter{}, DefaultOptions())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestRunApply_Parallel(t *testing.T) {
	opts := DefaultOptions()
	opts.Parallel = 2

	code, err := RunApply([]extensions.Extension{
		&mockExtension{id: "file:/a", name: "File /a", inSync: false},
		&mockExtension{id: "file:/b", name: "File /b", inSync: true},
		&mockExtension{id: "pkg:git", name: "Package git", inSync: false},
	}, &discardPrinter{}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRunApply_CriticalFailure(t *testing.T) {
	code, err := RunApply([]extensions.Extension{
		&criticalMock{
			mockExtension: mockExtension{id: "file:/a", name: "File /a", inSync: false, applyErr: fmt.Errorf("fail")},
			critical:      true,
		},
		&mockExtension{id: "file:/b", name: "File /b", inSync: false},
	}, &discardPrinter{}, DefaultOptions())
	if code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}
	if err == nil {
		t.Error("expected error for critical failure")
	}
}

func TestRunApply_WithTimeout(t *testing.T) {
	opts := DefaultOptions()
	opts.Timeout = 1 * time.Second

	code, err := RunApply([]extensions.Extension{
		&mockExtension{id: "file:/a", name: "File /a", inSync: true},
	}, &discardPrinter{}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}
