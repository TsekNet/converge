package autoedge

import (
	"context"
	"testing"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
)

type mockExt struct {
	id   string
	name string
}

func (m *mockExt) ID() string                                          { return m.id }
func (m *mockExt) Check(_ context.Context) (*extensions.State, error)  { return nil, nil }
func (m *mockExt) Apply(_ context.Context) (*extensions.Result, error) { return nil, nil }
func (m *mockExt) String() string                                      { return m.name }

func mock(id string) *mockExt { return &mockExt{id: id, name: id} }

func TestAddAutoEdges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(g *graph.Graph)
		wantLayers int
		verify     func(t *testing.T, layers [][]extensions.Extension)
	}{
		{
			name: "service depends on same-name package",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("package:nginx"))
				g.AddNode(mock("service:nginx"))
			},
			wantLayers: 2,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				assertLayerID(t, layers[0], 0, "package:nginx")
				assertLayerID(t, layers[1], 0, "service:nginx")
			},
		},
		{
			name: "file depends on parent directory",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("file:/etc/nginx"))
				g.AddNode(mock("file:/etc/nginx/nginx.conf"))
			},
			wantLayers: 2,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				assertLayerID(t, layers[0], 0, "file:/etc/nginx")
				assertLayerID(t, layers[1], 0, "file:/etc/nginx/nginx.conf")
			},
		},
		{
			name: "service depends on config file with matching name",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("file:/etc/nginx/nginx.conf"))
				g.AddNode(mock("service:nginx"))
			},
			wantLayers: 2,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				assertLayerID(t, layers[0], 0, "file:/etc/nginx/nginx.conf")
			},
		},
		{
			name: "unrelated resources stay in single layer",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("package:git"))
				g.AddNode(mock("service:nginx"))
				g.AddNode(mock("file:/etc/motd"))
			},
			wantLayers: 1,
			verify:     func(t *testing.T, layers [][]extensions.Extension) { t.Helper() },
		},
		{
			name: "skips edge that would create cycle",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("file:/etc/nginx/nginx.conf"))
				g.AddNode(mock("service:nginx"))
				g.AddEdge("file:/etc/nginx/nginx.conf", "service:nginx")
			},
			wantLayers: 2,
			verify:     func(t *testing.T, layers [][]extensions.Extension) { t.Helper() },
		},
		{
			name: "package and config file both precede service",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("package:nginx"))
				g.AddNode(mock("file:/etc/nginx/nginx.conf"))
				g.AddNode(mock("service:nginx"))
			},
			wantLayers: 2,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				if len(layers[0]) != 2 {
					t.Errorf("layer 0: got %d nodes, want 2", len(layers[0]))
				}
				if len(layers[1]) != 1 || layers[1][0].ID() != "service:nginx" {
					t.Errorf("layer 1: got %v, want [service:nginx]", idsOf(layers[1]))
				}
			},
		},
		{
			name: "parent dir creates third layer before config and service",
			setup: func(g *graph.Graph) {
				g.AddNode(mock("package:nginx"))
				g.AddNode(mock("file:/etc/nginx"))
				g.AddNode(mock("file:/etc/nginx/nginx.conf"))
				g.AddNode(mock("service:nginx"))
			},
			wantLayers: 3,
			verify: func(t *testing.T, layers [][]extensions.Extension) {
				t.Helper()
				if len(layers[0]) != 2 {
					t.Errorf("layer 0: got %d nodes, want 2", len(layers[0]))
				}
				if len(layers[1]) != 1 || layers[1][0].ID() != "file:/etc/nginx/nginx.conf" {
					t.Errorf("layer 1: got %v, want [file:/etc/nginx/nginx.conf]", idsOf(layers[1]))
				}
				if len(layers[2]) != 1 || layers[2][0].ID() != "service:nginx" {
					t.Errorf("layer 2: got %v, want [service:nginx]", idsOf(layers[2]))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := graph.New()
			tt.setup(g)

			if err := AddAutoEdges(g); err != nil {
				t.Fatalf("AddAutoEdges: %v", err)
			}

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

// helpers

func idsOf(exts []extensions.Extension) []string {
	ids := make([]string, len(exts))
	for i, e := range exts {
		ids[i] = e.ID()
	}
	return ids
}

func assertLayerID(t *testing.T, layer []extensions.Extension, idx int, wantID string) {
	t.Helper()
	if idx >= len(layer) {
		t.Fatalf("layer has %d nodes, want index %d", len(layer), idx)
	}
	if layer[idx].ID() != wantID {
		t.Errorf("layer[%d]: got %q, want %q", idx, layer[idx].ID(), wantID)
	}
}
