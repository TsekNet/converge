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
			name: "cycle detected",
			setup: func(g *Graph) {
				g.AddNode(mock("a", "A"))
				g.AddNode(mock("b", "B"))
				g.AddEdge("a", "b")
			},
			from:    "b",
			to:      "a",
			wantErr: true,
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
