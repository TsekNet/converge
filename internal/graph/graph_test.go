package graph

import (
	"context"
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
	g := New()
	pkg := mock("package:nginx", "Package nginx")

	if err := g.AddNode(pkg); err != nil {
		t.Fatalf("AddNode: %v", err)
	}

	got := g.Node("package:nginx")
	if got == nil {
		t.Fatal("Node not found after AddNode")
	}
	if got.Ext.ID() != "package:nginx" {
		t.Errorf("got ID %q, want %q", got.Ext.ID(), "package:nginx")
	}
}

func TestAddNodeDuplicate(t *testing.T) {
	g := New()
	pkg := mock("package:nginx", "Package nginx")

	if err := g.AddNode(pkg); err != nil {
		t.Fatalf("first AddNode: %v", err)
	}
	if err := g.AddNode(pkg); err == nil {
		t.Fatal("expected error on duplicate AddNode, got nil")
	}
}

func TestAddEdge(t *testing.T) {
	g := New()
	pkg := mock("package:nginx", "Package nginx")
	svc := mock("service:nginx", "Service nginx")

	g.AddNode(pkg)
	g.AddNode(svc)

	// service depends on package
	if err := g.AddEdge("service:nginx", "package:nginx"); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
}

func TestAddEdgeCycleDetection(t *testing.T) {
	g := New()
	a := mock("a", "A")
	b := mock("b", "B")

	g.AddNode(a)
	g.AddNode(b)
	g.AddEdge("a", "b")

	if err := g.AddEdge("b", "a"); err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestAddEdgeMissingNode(t *testing.T) {
	g := New()
	a := mock("a", "A")
	g.AddNode(a)

	if err := g.AddEdge("a", "nonexistent"); err == nil {
		t.Fatal("expected error for missing dependency node, got nil")
	}
	if err := g.AddEdge("nonexistent", "a"); err == nil {
		t.Fatal("expected error for missing dependent node, got nil")
	}
}

func TestTopologicalLayersLinear(t *testing.T) {
	// package -> file -> service (linear chain)
	g := New()
	pkg := mock("package:nginx", "Package nginx")
	file := mock("file:/etc/nginx/nginx.conf", "File /etc/nginx/nginx.conf")
	svc := mock("service:nginx", "Service nginx")

	g.AddNode(pkg)
	g.AddNode(file)
	g.AddNode(svc)
	g.AddEdge("file:/etc/nginx/nginx.conf", "package:nginx")
	g.AddEdge("service:nginx", "file:/etc/nginx/nginx.conf")

	layers, err := g.TopologicalLayers()
	if err != nil {
		t.Fatalf("TopologicalLayers: %v", err)
	}

	if len(layers) != 3 {
		t.Fatalf("got %d layers, want 3", len(layers))
	}

	// Layer 0: package (no deps)
	if len(layers[0]) != 1 || layers[0][0].ID() != "package:nginx" {
		t.Errorf("layer 0: got %v, want [package:nginx]", idsOf(layers[0]))
	}
	// Layer 1: file (depends on package)
	if len(layers[1]) != 1 || layers[1][0].ID() != "file:/etc/nginx/nginx.conf" {
		t.Errorf("layer 1: got %v, want [file:/etc/nginx/nginx.conf]", idsOf(layers[1]))
	}
	// Layer 2: service (depends on file)
	if len(layers[2]) != 1 || layers[2][0].ID() != "service:nginx" {
		t.Errorf("layer 2: got %v, want [service:nginx]", idsOf(layers[2]))
	}
}

func TestTopologicalLayersDiamond(t *testing.T) {
	// Diamond: A depends on B and C, both B and C depend on D
	//     D
	//    / \
	//   B   C
	//    \ /
	//     A
	g := New()
	g.AddNode(mock("d", "D"))
	g.AddNode(mock("b", "B"))
	g.AddNode(mock("c", "C"))
	g.AddNode(mock("a", "A"))

	g.AddEdge("b", "d")
	g.AddEdge("c", "d")
	g.AddEdge("a", "b")
	g.AddEdge("a", "c")

	layers, err := g.TopologicalLayers()
	if err != nil {
		t.Fatalf("TopologicalLayers: %v", err)
	}

	if len(layers) != 3 {
		t.Fatalf("got %d layers, want 3", len(layers))
	}

	// Layer 0: D (no deps)
	if len(layers[0]) != 1 || layers[0][0].ID() != "d" {
		t.Errorf("layer 0: got %v, want [d]", idsOf(layers[0]))
	}
	// Layer 1: B and C (both depend only on D)
	ids1 := idsOf(layers[1])
	if len(ids1) != 2 {
		t.Fatalf("layer 1: got %d nodes, want 2", len(ids1))
	}
	if !contains(ids1, "b") || !contains(ids1, "c") {
		t.Errorf("layer 1: got %v, want [b, c]", ids1)
	}
	// Layer 2: A (depends on B and C)
	if len(layers[2]) != 1 || layers[2][0].ID() != "a" {
		t.Errorf("layer 2: got %v, want [a]", idsOf(layers[2]))
	}
}

func TestTopologicalLayersNoEdges(t *testing.T) {
	g := New()
	g.AddNode(mock("a", "A"))
	g.AddNode(mock("b", "B"))
	g.AddNode(mock("c", "C"))

	layers, err := g.TopologicalLayers()
	if err != nil {
		t.Fatalf("TopologicalLayers: %v", err)
	}

	// All nodes have no deps: single layer
	if len(layers) != 1 {
		t.Fatalf("got %d layers, want 1", len(layers))
	}
	if len(layers[0]) != 3 {
		t.Errorf("layer 0: got %d nodes, want 3", len(layers[0]))
	}
}

func TestTopologicalLayersEmpty(t *testing.T) {
	g := New()
	layers, err := g.TopologicalLayers()
	if err != nil {
		t.Fatalf("TopologicalLayers: %v", err)
	}
	if len(layers) != 0 {
		t.Errorf("got %d layers, want 0", len(layers))
	}
}

func TestNodes(t *testing.T) {
	g := New()
	g.AddNode(mock("a", "A"))
	g.AddNode(mock("b", "B"))

	nodes := g.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
}

func TestOrderedExtensions(t *testing.T) {
	g := New()
	g.AddNode(mock("a", "A"))
	g.AddNode(mock("b", "B"))

	exts := g.OrderedExtensions()
	if len(exts) != 2 {
		t.Fatalf("got %d extensions, want 2", len(exts))
	}
	if exts[0].ID() != "a" || exts[1].ID() != "b" {
		t.Errorf("wrong order: got [%s, %s], want [a, b]", exts[0].ID(), exts[1].ID())
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
