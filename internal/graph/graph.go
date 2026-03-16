// Package graph provides a directed acyclic graph (DAG) of converge resources.
// Resources are vertices, dependency relationships are edges. The graph
// supports topological layer computation for parallel execution.
//
// Edge direction: AddEdge(fromID, toID) means fromID depends on toID.
// toID must complete before fromID starts.
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
// In-degree and adjacency are tracked incrementally for O(V+E)
// topological sorting, avoiding per-node parent queries.
type Graph struct {
	d        *dag.DAG
	nodes    map[string]*Node
	order    []string            // insertion order for deterministic iteration
	inDegree map[string]int      // number of dependencies per node
	children map[string][]string // children[id] = nodes that depend on id
}

// New creates an empty resource graph.
func New() *Graph {
	return &Graph{
		d:        dag.NewDAG(),
		nodes:    make(map[string]*Node),
		inDegree: make(map[string]int),
		children: make(map[string][]string),
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
	g.inDegree[id] = 0
	g.d.AddVertexByID(id, node)
	return nil
}

// AddEdge declares that fromID depends on toID (toID must run before fromID).
// Returns an error if either node is missing or the edge would create a cycle.
// Internally, heimdalr/dag uses parent->child direction, so we pass (toID, fromID).
func (g *Graph) AddEdge(fromID, toID string) error {
	if _, ok := g.nodes[fromID]; !ok {
		return fmt.Errorf("resource %q not found", fromID)
	}
	if _, ok := g.nodes[toID]; !ok {
		return fmt.Errorf("resource %q not found", toID)
	}
	if err := g.d.AddEdge(toID, fromID); err != nil {
		return err
	}
	// Track incrementally for O(V+E) topological sort.
	g.inDegree[fromID]++
	g.children[toID] = append(g.children[toID], fromID)
	return nil
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

// OrderedExtensions returns extensions in insertion order.
func (g *Graph) OrderedExtensions() []extensions.Extension {
	out := make([]extensions.Extension, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, g.nodes[id].Ext)
	}
	return out
}

// Flatten returns all extensions in topological order (flattened layers).
func (g *Graph) Flatten() ([]extensions.Extension, error) {
	layers, err := g.TopologicalLayers()
	if err != nil {
		return nil, err
	}
	var all []extensions.Extension
	for _, layer := range layers {
		all = append(all, layer...)
	}
	return all, nil
}

// TopologicalLayers returns resources grouped by dependency depth.
// Layer 0 contains resources with no dependencies. Layer N contains
// resources whose dependencies are all in layers < N. Resources within
// the same layer are independent and can run concurrently.
//
// Runs in O(V+E) using Kahn's algorithm with pre-computed in-degree.
func (g *Graph) TopologicalLayers() ([][]extensions.Extension, error) {
	if len(g.nodes) == 0 {
		return nil, nil
	}

	// Copy in-degree map (modified during sort).
	deg := make(map[string]int, len(g.nodes))
	for id, d := range g.inDegree {
		deg[id] = d
	}

	// Seed queue with zero-dependency nodes.
	queue := make([]string, 0, len(g.nodes))
	for id, d := range deg {
		if d == 0 {
			queue = append(queue, id)
		}
	}

	var layers [][]extensions.Extension
	placed := 0

	for len(queue) > 0 {
		layer := make([]extensions.Extension, 0, len(queue))
		var nextQueue []string

		for _, id := range queue {
			layer = append(layer, g.nodes[id].Ext)
			placed++
			for _, child := range g.children[id] {
				deg[child]--
				if deg[child] == 0 {
					nextQueue = append(nextQueue, child)
				}
			}
		}

		layers = append(layers, layer)
		queue = nextQueue
	}

	if placed != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected: placed %d of %d nodes", placed, len(g.nodes))
	}

	return layers, nil
}
