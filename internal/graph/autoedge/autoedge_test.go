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

func TestServiceDependsOnPackage(t *testing.T) {
	g := graph.New()
	g.AddNode(mock("package:nginx"))
	g.AddNode(mock("service:nginx"))

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(layers))
	}
	if layers[0][0].ID() != "package:nginx" {
		t.Errorf("layer 0: got %s, want package:nginx", layers[0][0].ID())
	}
	if layers[1][0].ID() != "service:nginx" {
		t.Errorf("layer 1: got %s, want service:nginx", layers[1][0].ID())
	}
}

func TestFileDependsOnParentDir(t *testing.T) {
	g := graph.New()
	g.AddNode(mock("file:/etc/nginx"))
	g.AddNode(mock("file:/etc/nginx/nginx.conf"))

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(layers))
	}
	if layers[0][0].ID() != "file:/etc/nginx" {
		t.Errorf("layer 0: got %s, want file:/etc/nginx", layers[0][0].ID())
	}
	if layers[1][0].ID() != "file:/etc/nginx/nginx.conf" {
		t.Errorf("layer 1: got %s, want file:/etc/nginx/nginx.conf", layers[1][0].ID())
	}
}

func TestServiceDependsOnConfigFile(t *testing.T) {
	g := graph.New()
	g.AddNode(mock("file:/etc/nginx/nginx.conf"))
	g.AddNode(mock("service:nginx"))

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(layers))
	}
	if layers[0][0].ID() != "file:/etc/nginx/nginx.conf" {
		t.Errorf("layer 0: got %s, want file:/etc/nginx/nginx.conf", layers[0][0].ID())
	}
}

func TestNoAutoEdgesForUnrelatedResources(t *testing.T) {
	g := graph.New()
	g.AddNode(mock("package:git"))
	g.AddNode(mock("service:nginx"))
	g.AddNode(mock("file:/etc/motd"))

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	// All unrelated: should be in one layer.
	if len(layers) != 1 {
		t.Fatalf("got %d layers, want 1 (all independent)", len(layers))
	}
}

func TestAutoEdgeSkipsCycles(t *testing.T) {
	// Pre-existing edge: file depends on service (unusual but valid).
	// Auto-edge would try service -> file, creating a cycle. Should skip.
	g := graph.New()
	g.AddNode(mock("file:/etc/nginx/nginx.conf"))
	g.AddNode(mock("service:nginx"))
	g.AddEdge("file:/etc/nginx/nginx.conf", "service:nginx")

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	// Should still be valid (2 layers), not error on cycle.
	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(layers))
	}
}

func TestFullChain(t *testing.T) {
	// Auto-edges create:
	//   service:nginx -> package:nginx  (serviceToPackage)
	//   service:nginx -> file:/etc/nginx/nginx.conf  (serviceToConfigFile)
	// package and file are independent (layer 0), service depends on both (layer 1).
	g := graph.New()
	g.AddNode(mock("package:nginx"))
	g.AddNode(mock("file:/etc/nginx/nginx.conf"))
	g.AddNode(mock("service:nginx"))

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(layers))
	}
	// Layer 0: package and file (independent)
	if len(layers[0]) != 2 {
		t.Errorf("layer 0: got %d nodes, want 2", len(layers[0]))
	}
	// Layer 1: service (depends on both)
	if len(layers[1]) != 1 || layers[1][0].ID() != "service:nginx" {
		t.Errorf("layer 1: got %v, want [service:nginx]", layers[1])
	}
}

func TestFullChainThreeLayers(t *testing.T) {
	// With a parent dir file, we get 3 layers:
	//   file:/etc/nginx/nginx.conf -> file:/etc/nginx (fileToParentDir)
	//   service:nginx -> package:nginx (serviceToPackage)
	//   service:nginx -> file:/etc/nginx/nginx.conf (serviceToConfigFile)
	g := graph.New()
	g.AddNode(mock("package:nginx"))
	g.AddNode(mock("file:/etc/nginx"))
	g.AddNode(mock("file:/etc/nginx/nginx.conf"))
	g.AddNode(mock("service:nginx"))

	if err := AddAutoEdges(g); err != nil {
		t.Fatalf("AddAutoEdges: %v", err)
	}

	layers, _ := g.TopologicalLayers()
	if len(layers) != 3 {
		t.Fatalf("got %d layers, want 3", len(layers))
	}
	// Layer 0: package:nginx and file:/etc/nginx (no deps)
	if len(layers[0]) != 2 {
		t.Errorf("layer 0: got %d nodes, want 2", len(layers[0]))
	}
	// Layer 1: file:/etc/nginx/nginx.conf (depends on parent dir)
	if len(layers[1]) != 1 || layers[1][0].ID() != "file:/etc/nginx/nginx.conf" {
		t.Errorf("layer 1: got %v, want [file:/etc/nginx/nginx.conf]", layers[1])
	}
	// Layer 2: service:nginx (depends on file and package)
	if len(layers[2]) != 1 || layers[2][0].ID() != "service:nginx" {
		t.Errorf("layer 2: got %v, want [service:nginx]", layers[2])
	}
}
