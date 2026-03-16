// Package graph provides a directed acyclic graph (DAG) of converge resources.
// Resources are vertices, dependency relationships are edges. The graph
// supports topological layer computation for parallel execution.
package graph

import (
	"fmt"

	"github.com/TsekNet/converge/extensions"
	"github.com/heimdalr/dag"
)

// Node wraps an Extension with DAG metadata.
type Node struct {
	Ext extensions.Extension
}

// ID implements dag.IDInterface for the heimdalr/dag library.
func (n *Node) ID() string {
	return n.Ext.ID()
}

// Graph holds all resource nodes and their dependency edges.
type Graph struct {
	d     *dag.DAG
	nodes map[string]*Node
	order []string // insertion order for deterministic iteration
}

// New creates an empty resource graph.
func New() *Graph {
	return &Graph{
		d:     dag.NewDAG(),
		nodes: make(map[string]*Node),
	}
}

// AddNode adds a resource to the graph. Returns an error if a resource
// with the same ID already exists.
func (g *Graph) AddNode(ext extensions.Extension) error {
	id := ext.ID()
	if _, exists := g.nodes[id]; exists {
		return fmt.Errorf("duplicate resource: %s", id)
	}

	node := &Node{Ext: ext}
	g.nodes[id] = node
	g.order = append(g.order, id)
	g.d.AddVertexByID(id, node)
	return nil
}

// AddEdge declares that the resource identified by fromID depends on the
// resource identified by toID (toID must run before fromID). Returns an
// error if either node is missing or the edge would create a cycle.
func (g *Graph) AddEdge(fromID, toID string) error {
	if _, ok := g.nodes[fromID]; !ok {
		return fmt.Errorf("resource %q not found", fromID)
	}
	if _, ok := g.nodes[toID]; !ok {
		return fmt.Errorf("resource %q not found", toID)
	}
	return g.d.AddEdge(toID, fromID)
}

// Node returns the node with the given ID, or nil if not found.
func (g *Graph) Node(id string) *Node {
	return g.nodes[id]
}

// Nodes returns all nodes in the graph (unordered).
func (g *Graph) Nodes() []*Node {
	out := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n)
	}
	return out
}

// AllExtensions returns all extensions in the graph (unordered).
func (g *Graph) AllExtensions() []extensions.Extension {
	out := make([]extensions.Extension, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n.Ext)
	}
	return out
}

// OrderedExtensions returns extensions in insertion order.
func (g *Graph) OrderedExtensions() []extensions.Extension {
	out := make([]extensions.Extension, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, g.nodes[id].Ext)
	}
	return out
}

// TopologicalLayers returns resources grouped by dependency depth.
// Layer 0 contains resources with no dependencies. Layer N contains
// resources whose dependencies are all in layers < N. Resources within
// the same layer are independent and can run concurrently.
func (g *Graph) TopologicalLayers() ([][]extensions.Extension, error) {
	if len(g.nodes) == 0 {
		return nil, nil
	}

	// Compute in-degree for each node.
	inDegree := make(map[string]int, len(g.nodes))
	// children[id] = list of nodes that depend on id (id must run first).
	children := make(map[string][]string, len(g.nodes))

	for id := range g.nodes {
		inDegree[id] = 0
	}

	// For each node, find its parents (dependencies).
	for id := range g.nodes {
		parents, err := g.d.GetParents(id)
		if err != nil {
			return nil, fmt.Errorf("getting parents of %s: %w", id, err)
		}
		inDegree[id] = len(parents)
		for pid := range parents {
			children[pid] = append(children[pid], id)
		}
	}

	var layers [][]extensions.Extension

	// BFS by layers (Kahn's algorithm variant).
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	for len(queue) > 0 {
		layer := make([]extensions.Extension, 0, len(queue))
		var nextQueue []string

		for _, id := range queue {
			layer = append(layer, g.nodes[id].Ext)
			for _, child := range children[id] {
				inDegree[child]--
				if inDegree[child] == 0 {
					nextQueue = append(nextQueue, child)
				}
			}
		}

		layers = append(layers, layer)
		queue = nextQueue
	}

	// Sanity check: all nodes should be placed.
	placed := 0
	for _, l := range layers {
		placed += len(l)
	}
	if placed != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected: placed %d of %d nodes", placed, len(g.nodes))
	}

	return layers, nil
}
