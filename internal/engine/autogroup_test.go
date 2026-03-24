package engine

import (
	"context"
	"testing"

	"github.com/TsekNet/converge/extensions"
	extpkg "github.com/TsekNet/converge/extensions/pkg"
	"github.com/TsekNet/converge/internal/graph"
)

// testManager implements extpkg.PackageManager without batch support.
type testManager struct {
	name      string
	installed map[string]bool
}

func (m *testManager) Name() string { return m.name }
func (m *testManager) IsInstalled(_ context.Context, name string) (bool, error) {
	return m.installed[name], nil
}
func (m *testManager) Install(_ context.Context, name string) error {
	m.installed[name] = true
	return nil
}
func (m *testManager) Remove(_ context.Context, name string) error {
	delete(m.installed, name)
	return nil
}

// testBatchManager implements both PackageManager and BatchInstaller.
type testBatchManager struct {
	testManager
	batchInstallCalls [][]string
	batchRemoveCalls  [][]string
}

func (m *testBatchManager) InstallBatch(_ context.Context, names []string) error {
	cp := make([]string, len(names))
	copy(cp, names)
	m.batchInstallCalls = append(m.batchInstallCalls, cp)
	for _, n := range names {
		m.installed[n] = true
	}
	return nil
}

func (m *testBatchManager) RemoveBatch(_ context.Context, names []string) error {
	cp := make([]string, len(names))
	copy(cp, names)
	m.batchRemoveCalls = append(m.batchRemoveCalls, cp)
	for _, n := range names {
		delete(m.installed, n)
	}
	return nil
}

// fileMock is a non-package Extension for autoGroupLayer tests.
type fileMock struct{ id string }

func (f *fileMock) ID() string                                          { return f.id }
func (f *fileMock) Check(_ context.Context) (*extensions.State, error)  { return nil, nil }
func (f *fileMock) Apply(_ context.Context) (*extensions.Result, error) { return nil, nil }
func (f *fileMock) String() string                                      { return f.id }

func boolPtr(b bool) *bool { return &b }

func pkgNode(name, state, mgrName string, mgr extpkg.PackageManager) *graph.Node {
	return &graph.Node{
		Ext: &extpkg.Package{
			PkgName:     name,
			State:       state,
			Manager:     mgr,
			ManagerName: mgrName,
		},
	}
}

func TestPackageGroup_ID(t *testing.T) {
	t.Parallel()
	mgr := &testManager{name: "apt", installed: map[string]bool{}}
	pg := &PackageGroup{
		Packages: []*extpkg.Package{
			{PkgName: "zsh", Manager: mgr, ManagerName: "apt", State: "present"},
			{PkgName: "curl", Manager: mgr, ManagerName: "apt", State: "present"},
			{PkgName: "git", Manager: mgr, ManagerName: "apt", State: "present"},
		},
		Manager: mgr,
		State:   "present",
	}
	got := pg.ID()
	want := "packages:curl,git,zsh"
	if got != want {
		t.Errorf("ID() = %q, want %q", got, want)
	}
}

func TestPackageGroup_String(t *testing.T) {
	t.Parallel()
	mgr := &testManager{name: "apt", installed: map[string]bool{}}
	pg := &PackageGroup{
		Packages: []*extpkg.Package{
			{PkgName: "zsh", Manager: mgr, ManagerName: "apt", State: "present"},
			{PkgName: "curl", Manager: mgr, ManagerName: "apt", State: "present"},
		},
		Manager: mgr,
		State:   "present",
	}
	got := pg.String()
	want := "Packages [curl, zsh]"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPackageGroup_IsCritical(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		critical []bool
		want     bool
	}{
		{"none critical", []bool{false, false}, false},
		{"one critical", []bool{false, true}, true},
		{"all critical", []bool{true, true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := &testManager{name: "apt", installed: map[string]bool{}}
			var pkgs []*extpkg.Package
			for i, c := range tt.critical {
				pkgs = append(pkgs, &extpkg.Package{
					PkgName:  "pkg" + string(rune('a'+i)),
					Critical: c,
					Manager:  mgr,
					State:    "present",
				})
			}
			pg := &PackageGroup{Packages: pkgs, Manager: mgr, State: "present"}
			if got := pg.IsCritical(); got != tt.want {
				t.Errorf("IsCritical() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPackageGroup_Check(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		installed map[string]bool
		state     string
		wantSync  bool
	}{
		{
			"all in sync",
			map[string]bool{"curl": true, "git": true},
			"present",
			true,
		},
		{
			"one out of sync",
			map[string]bool{"curl": true, "git": false},
			"present",
			false,
		},
		{
			"absent all in sync",
			map[string]bool{"curl": false, "git": false},
			"absent",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := &testManager{name: "apt", installed: tt.installed}
			pkgs := []*extpkg.Package{
				{PkgName: "curl", State: tt.state, Manager: mgr, ManagerName: "apt"},
				{PkgName: "git", State: tt.state, Manager: mgr, ManagerName: "apt"},
			}
			pg := &PackageGroup{Packages: pkgs, Manager: mgr, State: tt.state}
			state, err := pg.Check(context.Background())
			if err != nil {
				t.Fatalf("Check() error: %v", err)
			}
			if state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", state.InSync, tt.wantSync)
			}
			if !tt.wantSync && len(state.Changes) == 0 {
				t.Error("expected changes when out of sync")
			}
		})
	}
}

func TestPackageGroup_Apply_BatchInstaller(t *testing.T) {
	t.Parallel()
	mgr := &testBatchManager{
		testManager: testManager{name: "apt", installed: map[string]bool{}},
	}
	pkgs := []*extpkg.Package{
		{PkgName: "curl", State: "present", Manager: mgr, ManagerName: "apt"},
		{PkgName: "git", State: "present", Manager: mgr, ManagerName: "apt"},
	}
	pg := &PackageGroup{Packages: pkgs, Manager: mgr, State: "present"}

	result, err := pg.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if result.Status != extensions.StatusChanged {
		t.Errorf("Status = %v, want StatusChanged", result.Status)
	}
	if !result.Changed {
		t.Error("Changed = false, want true")
	}
	if len(mgr.batchInstallCalls) != 1 {
		t.Fatalf("expected 1 batch install call, got %d", len(mgr.batchInstallCalls))
	}
	got := mgr.batchInstallCalls[0]
	if len(got) != 2 {
		t.Fatalf("expected 2 packages in batch call, got %d", len(got))
	}
}

func TestPackageGroup_Apply_Individual(t *testing.T) {
	t.Parallel()
	mgr := &testManager{name: "apt", installed: map[string]bool{}}
	pkgs := []*extpkg.Package{
		{PkgName: "curl", State: "present", Manager: mgr, ManagerName: "apt"},
		{PkgName: "git", State: "present", Manager: mgr, ManagerName: "apt"},
	}
	pg := &PackageGroup{Packages: pkgs, Manager: mgr, State: "present"}

	result, err := pg.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if result.Status != extensions.StatusChanged {
		t.Errorf("Status = %v, want StatusChanged", result.Status)
	}
	if !mgr.installed["curl"] || !mgr.installed["git"] {
		t.Errorf("expected both packages installed, got %v", mgr.installed)
	}
}

func TestPackageGroup_Apply_AllInSync(t *testing.T) {
	t.Parallel()
	mgr := &testManager{name: "apt", installed: map[string]bool{"curl": true, "git": true}}
	pkgs := []*extpkg.Package{
		{PkgName: "curl", State: "present", Manager: mgr, ManagerName: "apt"},
		{PkgName: "git", State: "present", Manager: mgr, ManagerName: "apt"},
	}
	pg := &PackageGroup{Packages: pkgs, Manager: mgr, State: "present"}

	result, err := pg.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if result.Status != extensions.StatusOK {
		t.Errorf("Status = %v, want StatusOK", result.Status)
	}
	if result.Changed {
		t.Error("Changed = true, want false")
	}
}

func TestAutoGroupLayer(t *testing.T) {
	t.Parallel()
	mgr1 := &testManager{name: "apt", installed: map[string]bool{}}
	mgr2 := &testManager{name: "brew", installed: map[string]bool{}}

	tests := []struct {
		name       string
		layer      []*graph.Node
		wantCount  int
		wantGroups int // number of PackageGroup in result
	}{
		{
			name: "mixed types",
			layer: []*graph.Node{
				pkgNode("curl", "present", "apt", mgr1),
				pkgNode("git", "present", "apt", mgr1),
				{Ext: &fileMock{id: "file:/etc/hosts"}},
			},
			wantCount:  2, // 1 PackageGroup + 1 fileMock
			wantGroups: 1,
		},
		{
			name: "single package not grouped",
			layer: []*graph.Node{
				pkgNode("curl", "present", "apt", mgr1),
			},
			wantCount:  1,
			wantGroups: 0,
		},
		{
			name: "autogroup disabled",
			layer: []*graph.Node{
				{
					Ext:  &extpkg.Package{PkgName: "curl", State: "present", Manager: mgr1, ManagerName: "apt"},
					Meta: graph.NodeMeta{AutoGroup: boolPtr(false)},
				},
				pkgNode("git", "present", "apt", mgr1),
			},
			wantCount:  2, // curl ungrouped + git ungrouped (only 1 left, no group)
			wantGroups: 0,
		},
		{
			name: "different managers",
			layer: []*graph.Node{
				pkgNode("curl", "present", "apt", mgr1),
				pkgNode("git", "present", "brew", mgr2),
			},
			wantCount:  2, // each is alone in its group, returned as-is
			wantGroups: 0,
		},
		{
			name: "different states",
			layer: []*graph.Node{
				pkgNode("curl", "present", "apt", mgr1),
				pkgNode("git", "absent", "apt", mgr1),
			},
			wantCount:  2,
			wantGroups: 0,
		},
		{
			name:       "empty layer",
			layer:      []*graph.Node{},
			wantCount:  0,
			wantGroups: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := autoGroupLayer(tt.layer)
			if len(result) != tt.wantCount {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantCount)
				for i, ext := range result {
					t.Logf("  result[%d]: %T %s", i, ext, ext.ID())
				}
			}
			groups := 0
			for _, ext := range result {
				if _, ok := ext.(*PackageGroup); ok {
					groups++
				}
			}
			if groups != tt.wantGroups {
				t.Errorf("PackageGroup count = %d, want %d", groups, tt.wantGroups)
			}
		})
	}
}
