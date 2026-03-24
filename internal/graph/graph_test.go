package graph

import (
	"context"
	"fmt"
	"testing"

	"github.com/TsekNet/converge/extensions"
)

// mockExtension implements extensions.Extension for testing.
type mockExtension struct {
	id   string
	name string
}

func (m *mockExtension) ID() string                                    { return m.id }
func (m *mockExtension) Check(_ context.Context) (*extensions.State, error) { return nil, nil }
func (m *mockExtension) Apply(_ context.Context) (*extensions.Result, error) { return nil, nil }
func (m *mockExtension) String() string                                { return m.name }

func mock(id, name string) extensions.Extension {
	return &mockExtension{id: id, name: name}
}

func TestAddNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ids     []string // nodes to add in sequence
		wantErr bool     // whether the last AddNode should error
	}{
		{
			name:    "single node is retrievable",
			ids:     []string{"package:nginx"},
			wantErr: false,
		},
		{
			name:    "duplicate node returns error",
			ids:     []string{"package:nginx", "package:nginx"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			var err error
			for _, id := range tt.ids {
				err = g.AddNode(mock(id, id))
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("AddNode error = %v, wantErr = %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				last := tt.ids[len(tt.ids)-1]
				got := g.Node(last)
				if got == nil {
					t.Fatal("Node not found after AddNode")
				}
				if got.Ext.ID() != last {
					t.Errorf("got ID %q, want %q", got.Ext.ID(), last)
				}
			}
		})
	}
}

func TestAddEdge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(g *Graph)
		from    string
		to      string
		wantErr bool
	}{
		{
			name: "valid edge between two nodes",
			setup: func(g *Graph) {
				g.AddNode(mock("package:nginx", "Package nginx"))
				g.AddNode(mock("service:nginx", "Service nginx"))
			},
			from:    "service:nginx",
			to:      "package:nginx",
			wantErr: false,
		},
		{
			name: "duplicate edge is silently ignored",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddEdge("a", "b")
			},
			from:    "a",
			to:      "b",
			wantErr: false,
		},
		{
			name: "missing dependency node",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
			},
			from:    "a",
			to:      "nonexistent",
			wantErr: true,
		},
		{
			name: "missing dependent node",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
			},
			from:    "nonexistent",
			to:      "a",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			tt.setup(g)

			err := g.AddEdge(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddEdge(%q, %q) error = %v, wantErr = %v", tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestTopologicalLayers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(g *Graph)
		wantLayers int
		verify     func(t *testing.T, layers [][]extensions.Extension)
	}{
		{
			name: "linear chain produces one node per layer",
			setup: func(g *Graph) {
				g.AddNode(mock("package:nginx", "Package nginx"))
				g.AddNode(mock("file:/etc/nginx/nginx.conf", "File /etc/nginx/nginx.conf"))
				g.AddNode(mock("service:nginx", "Service nginx"))
				g.AddEdge("file:/etc/nginx/nginx.conf", "package:nginx")
				g.AddEdge("service:nginx", "file:/etc/nginx/nginx.conf")
			},
			wantLayers: 3,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				assertLayerIDs(t, layers[0], "package:nginx")
				assertLayerIDs(t, layers[1], "file:/etc/nginx/nginx.conf")
				assertLayerIDs(t, layers[2], "service:nginx")
			},
		},
		{
			name: "diamond graph merges independent nodes into one layer",
			setup: func(g *Graph) {
				g.AddNode(mock("d", "D"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
				g.AddNode(mock("a", "A"))
				g.AddEdge("b", "d")
				g.AddEdge("c", "d")
				g.AddEdge("a", "b")
				g.AddEdge("a", "c")
			},
			wantLayers: 3,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				assertLayerIDs(t, layers[0], "d")
				assertLayerContains(t, layers[1], "b", "c")
				assertLayerIDs(t, layers[2], "a")
			},
		},
		{
			name: "no edges puts all nodes in single layer",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
			},
			wantLayers: 1,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				if len(layers[0]) != 3 {
					t.Errorf("layer 0: got %d nodes, want 3", len(layers[0]))
				}
			},
		},
		{
			name:       "empty graph returns no layers",
			setup:      func(g *Graph) {},
			wantLayers: 0,
			verify:     func(t *testing.T, layers [][]extensions.Extension) { t.Helper() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			tt.setup(g)

			layers, err := g.TopologicalLayers()
			if err != nil {
				t.Fatalf("TopologicalLayers: %v", err)
			}
			if len(layers) != tt.wantLayers {
				t.Fatalf("got %d layers, want %d", len(layers), tt.wantLayers)
			}
			tt.verify(t, layers)
		})
	}
}

func TestTopologicalLayers_CycleDetection(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode(mock("a", "A"))
	g.AddNode(mock("b", "B"))
	g.AddEdge("a", "b")
	g.AddEdge("b", "a") // creates cycle, accepted by AddEdge (lazy detection)

	_, err := g.TopologicalLayers()
	if err == nil {
		t.Fatal("expected cycle error from TopologicalLayers, got nil")
	}
}

func BenchmarkTopologicalLayers_2000(b *testing.B) {
	// 2000 nodes in a linear chain: 0 -> 1 -> 2 -> ... -> 1999.
	g := New()
	for i := 0; i < 2000; i++ {
		id := fmt.Sprintf("resource:%d", i)
		g.AddNode(mock(id, id))
	}
	for i := 1; i < 2000; i++ {
		from := fmt.Sprintf("resource:%d", i)
		to := fmt.Sprintf("resource:%d", i-1)
		g.AddEdge(from, to)
	}

	b.ResetTimer()
	for range b.N {
		g.TopologicalLayers()
	}
}

func BenchmarkTopologicalLayers_2000_Wide(b *testing.B) {
	// 2000 nodes with 10 layers of 200, each depending on all nodes in the previous layer.
	g := New()
	for i := 0; i < 2000; i++ {
		id := fmt.Sprintf("resource:%d", i)
		g.AddNode(mock(id, id))
	}
	for layer := 1; layer < 10; layer++ {
		for i := 0; i < 200; i++ {
			from := fmt.Sprintf("resource:%d", layer*200+i)
			to := fmt.Sprintf("resource:%d", (layer-1)*200) // depend on first node in prev layer
			g.AddEdge(from, to)
		}
	}

	b.ResetTimer()
	for range b.N {
		g.TopologicalLayers()
	}
}

func TestAddEdge_SelfDependency(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode(mock("a", "A"))

	err := g.AddEdge("a", "a")
	if err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}
}

func TestNodes_InsertionOrder(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode(mock("x", "X"))
	g.AddNode(mock("y", "Y"))
	g.AddNode(mock("z", "Z"))

	nodes := g.Nodes()
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}

	wantIDs := []string{"x", "y", "z"}
	for i, n := range nodes {
		if n.Ext.ID() != wantIDs[i] {
			t.Errorf("Nodes()[%d].Ext.ID() = %q, want %q", i, n.Ext.ID(), wantIDs[i])
		}
	}
}

func TestOrderedExtensions(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode(mock("alpha", "Alpha"))
	g.AddNode(mock("beta", "Beta"))
	g.AddNode(mock("gamma", "Gamma"))

	exts := g.OrderedExtensions()
	if len(exts) != 3 {
		t.Fatalf("got %d extensions, want 3", len(exts))
	}

	wantIDs := []string{"alpha", "beta", "gamma"}
	for i, ext := range exts {
		if ext.ID() != wantIDs[i] {
			t.Errorf("OrderedExtensions()[%d].ID() = %q, want %q", i, ext.ID(), wantIDs[i])
		}
	}
}

func TestFlatten(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(g *Graph)
		verify  func(t *testing.T, exts []extensions.Extension)
		wantErr bool
	}{
		{
			name: "linear chain returns topological order",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
				g.AddEdge("b", "a")
				g.AddEdge("c", "b")
			},
			verify: func(t *testing.T, exts []extensions.Extension) {
				t.Helper()
				ids := idsOf(exts)
				want := []string{"a", "b", "c"}
				if len(ids) != len(want) {
					t.Fatalf("got %v, want %v", ids, want)
				}
				for i, id := range want {
					if ids[i] != id {
						t.Errorf("Flatten()[%d] = %q, want %q", i, ids[i], id)
					}
				}
			},
		},
		{
			name: "diamond returns valid topological order",
			setup: func(g *Graph) {
				g.AddNode(mock("d", "D"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
				g.AddNode(mock("a", "A"))
				g.AddEdge("b", "d")
				g.AddEdge("c", "d")
				g.AddEdge("a", "b")
				g.AddEdge("a", "c")
			},
			verify: func(t *testing.T, exts []extensions.Extension) {
				t.Helper()
				ids := idsOf(exts)
				if len(ids) != 4 {
					t.Fatalf("got %d extensions, want 4", len(ids))
				}
				// d must come before b and c; b and c must come before a.
				pos := make(map[string]int, len(ids))
				for i, id := range ids {
					pos[id] = i
				}
				if pos["d"] >= pos["b"] {
					t.Errorf("d (pos %d) must come before b (pos %d)", pos["d"], pos["b"])
				}
				if pos["d"] >= pos["c"] {
					t.Errorf("d (pos %d) must come before c (pos %d)", pos["d"], pos["c"])
				}
				if pos["b"] >= pos["a"] {
					t.Errorf("b (pos %d) must come before a (pos %d)", pos["b"], pos["a"])
				}
				if pos["c"] >= pos["a"] {
					t.Errorf("c (pos %d) must come before a (pos %d)", pos["c"], pos["a"])
				}
			},
		},
		{
			name:  "empty graph returns nil",
			setup: func(g *Graph) {},
			verify: func(t *testing.T, exts []extensions.Extension) {
				t.Helper()
				if exts != nil {
					t.Errorf("got %v, want nil", exts)
				}
			},
		},
		{
			name: "cycle returns error",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddEdge("a", "b")
				g.AddEdge("b", "a")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			tt.setup(g)

			exts, err := g.Flatten()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Flatten() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				tt.verify(t, exts)
			}
		})
	}
}

func TestWouldCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(g *Graph)
		from  string
		to    string
		want  bool
	}{
		{
			name: "no cycle returns false",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddEdge("a", "b") // a depends on b
			},
			from: "a",
			to:   "b",
			want: false,
		},
		{
			name: "direct cycle returns true",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddEdge("a", "b") // a depends on b
			},
			from: "b",
			to:   "a",
			want: true,
		},
		{
			name: "existing reverse edge means cycle",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddEdge("b", "a") // b depends on a, so a -> children -> b
			},
			from: "a",
			to:   "b",
			want: true,
		},
		{
			name: "transitive cycle returns true",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
				g.AddEdge("b", "a") // b depends on a
				g.AddEdge("c", "b") // c depends on b
			},
			from: "a",
			to:   "c",
			want: true,
		},
		{
			name: "disconnected nodes returns false",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
			},
			from: "a",
			to:   "b",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			tt.setup(g)

			got := g.WouldCycle(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("WouldCycle(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestChildren(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode(mock("a", "A"))
	g.AddNode(mock("b", "B"))
	g.AddNode(mock("c", "C"))
	g.AddNode(mock("d", "D"))
	g.AddEdge("b", "a") // b depends on a
	g.AddEdge("c", "a") // c depends on a
	g.AddEdge("d", "b") // d depends on b

	got := g.Children("a")
	if len(got) != 2 {
		t.Fatalf("Children(a): got %d, want 2", len(got))
	}
	wantSet := map[string]bool{"b": true, "c": true}
	for _, id := range got {
		if !wantSet[id] {
			t.Errorf("unexpected child of a: %q", id)
		}
	}

	got = g.Children("b")
	if len(got) != 1 || got[0] != "d" {
		t.Errorf("Children(b) = %v, want [d]", got)
	}

	got = g.Children("d")
	if len(got) != 0 {
		t.Errorf("Children(d) = %v, want []", got)
	}
}

func TestTopologicalNodeLayers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(g *Graph)
		wantLayers int
		verify     func(t *testing.T, layers [][]*Node)
	}{
		{
			name: "linear chain produces one node per layer",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
				g.AddEdge("b", "a")
				g.AddEdge("c", "b")
			},
			wantLayers: 3,
			verify: func(t *testing.T, layers [][]*Node) {
				t.Helper()
				wantPerLayer := []string{"a", "b", "c"}
				for i, wantID := range wantPerLayer {
					if len(layers[i]) != 1 {
						t.Fatalf("layer %d: got %d nodes, want 1", i, len(layers[i]))
					}
					if layers[i][0].Ext.ID() != wantID {
						t.Errorf("layer %d: got %q, want %q", i, layers[i][0].Ext.ID(), wantID)
					}
				}
			},
		},
		{
			name: "diamond produces three layers",
			setup: func(g *Graph) {
				g.AddNode(mock("d", "D"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
				g.AddNode(mock("a", "A"))
				g.AddEdge("b", "d")
				g.AddEdge("c", "d")
				g.AddEdge("a", "b")
				g.AddEdge("a", "c")
			},
			wantLayers: 3,
			verify: func(t *testing.T, layers [][]*Node) {
				t.Helper()
				if layers[0][0].Ext.ID() != "d" {
					t.Errorf("layer 0: got %q, want d", layers[0][0].Ext.ID())
				}
				if len(layers[1]) != 2 {
					t.Fatalf("layer 1: got %d nodes, want 2", len(layers[1]))
				}
				midIDs := map[string]bool{
					layers[1][0].Ext.ID(): true,
					layers[1][1].Ext.ID(): true,
				}
				if !midIDs["b"] || !midIDs["c"] {
					t.Errorf("layer 1: got %v, want b and c", midIDs)
				}
				if layers[2][0].Ext.ID() != "a" {
					t.Errorf("layer 2: got %q, want a", layers[2][0].Ext.ID())
				}
			},
		},
		{
			name:       "empty graph returns nil",
			setup:      func(g *Graph) {},
			wantLayers: 0,
			verify:     func(t *testing.T, layers [][]*Node) { t.Helper() },
		},
		{
			name: "no edges puts all nodes in single layer",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddNode(mock("c", "C"))
			},
			wantLayers: 1,
			verify: func(t *testing.T, layers [][]*Node) {
				t.Helper()
				if len(layers[0]) != 3 {
					t.Errorf("layer 0: got %d nodes, want 3", len(layers[0]))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New()
			tt.setup(g)

			layers, err := g.TopologicalNodeLayers()
			if err != nil {
				t.Fatalf("TopologicalNodeLayers: %v", err)
			}
			if len(layers) != tt.wantLayers {
				t.Fatalf("got %d layers, want %d", len(layers), tt.wantLayers)
			}
			tt.verify(t, layers)
		})
	}
}

func TestSetMeta(t *testing.T) {
	t.Parallel()

	t.Run("sets meta on existing node", func(t *testing.T) {
		t.Parallel()

		g := New()
		g.AddNode(mock("a", "A"))

		meta := NodeMeta{Noop: true, Retry: 3, Limit: 1.5}
		g.SetMeta("a", meta)

		n := g.Node("a")
		if n == nil {
			t.Fatal("node not found")
		}
		if !n.Meta.Noop {
			t.Error("Meta.Noop: got false, want true")
		}
		if n.Meta.Retry != 3 {
			t.Errorf("Meta.Retry: got %d, want 3", n.Meta.Retry)
		}
		if n.Meta.Limit != 1.5 {
			t.Errorf("Meta.Limit: got %f, want 1.5", n.Meta.Limit)
		}
	})

	t.Run("nonexistent node is a no-op", func(t *testing.T) {
		t.Parallel()

		g := New()
		// Should not panic.
		g.SetMeta("nonexistent", NodeMeta{Noop: true})
	})
}

// helpers

func idsOf(exts []extensions.Extension) []string {
	ids := make([]string, len(exts))
	for i, e := range exts {
		ids[i] = e.ID()
	}
	return ids
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func assertLayerIDs(t *testing.T, layer []extensions.Extension, wantIDs ...string) {
	t.Helper()
	got := idsOf(layer)
	if len(got) != len(wantIDs) {
		t.Fatalf("layer: got %v, want %v", got, wantIDs)
	}
	for i, id := range wantIDs {
		if got[i] != id {
			t.Errorf("layer[%d]: got %q, want %q", i, got[i], id)
		}
	}
}

func assertLayerContains(t *testing.T, layer []extensions.Extension, wantIDs ...string) {
	t.Helper()
	got := idsOf(layer)
	if len(got) != len(wantIDs) {
		t.Fatalf("layer: got %v, want %v", got, wantIDs)
	}
	for _, id := range wantIDs {
		if !contains(got, id) {
			t.Errorf("layer %v missing %q", got, id)
		}
	}
}
