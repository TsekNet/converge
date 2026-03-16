// Package autoedge detects and adds implicit dependency edges between
// resources in a graph. For example, a service automatically depends on
// its package, and a file depends on its parent directory.
package autoedge

import (
	"path/filepath"
	"strings"

	"github.com/TsekNet/converge/internal/graph"
	"github.com/google/deck"
)

// rule is a function that inspects the graph and adds implicit edges.
type rule func(g *graph.Graph) error

var rules = []rule{
	serviceToPackage,
	fileToParentDir,
	serviceToConfigFile,
}

// autoEdgeDisabled returns true if the node has AutoEdge explicitly set to false.
func autoEdgeDisabled(g *graph.Graph, id string) bool {
	node := g.Node(id)
	if node == nil {
		return false
	}
	return node.Meta.AutoEdge != nil && !*node.Meta.AutoEdge
}

// AddAutoEdges applies all auto-edge rules to the graph.
// Nodes with AutoEdge=false in their meta are skipped.
// Edges that would create cycles are silently skipped (logged as warnings).
func AddAutoEdges(g *graph.Graph) error {
	for _, r := range rules {
		if err := r(g); err != nil {
			return err
		}
	}
	return nil
}

// serviceToPackage: service:X depends on package:X (name match).
func serviceToPackage(g *graph.Graph) error {
	for _, node := range g.Nodes() {
		id := node.Ext.ID()
		if !strings.HasPrefix(id, "service:") {
			continue
		}
		if autoEdgeDisabled(g, id) {
			continue
		}
		name := strings.TrimPrefix(id, "service:")
		depID := "package:" + name
		if g.Node(depID) == nil {
			continue
		}
		if autoEdgeDisabled(g, depID) {
			continue
		}
		tryAddEdge(g, id, depID)
	}
	return nil
}

// fileToParentDir: file:/a/b/c depends on file:/a/b (parent path match).
func fileToParentDir(g *graph.Graph) error {
	for _, node := range g.Nodes() {
		id := node.Ext.ID()
		if !strings.HasPrefix(id, "file:") {
			continue
		}
		if autoEdgeDisabled(g, id) {
			continue
		}
		path := strings.TrimPrefix(id, "file:")
		parentPath := filepath.Dir(path)
		parentID := "file:" + parentPath
		if parentID == id || g.Node(parentID) == nil {
			continue
		}
		if autoEdgeDisabled(g, parentID) {
			continue
		}
		tryAddEdge(g, id, parentID)
	}
	return nil
}

// serviceToConfigFile: service:X depends on files whose path contains the
// service name as a path component or base filename. Requires the service name
// to be at least 3 characters to avoid false positives with short names.
func serviceToConfigFile(g *graph.Graph) error {
	for _, svcNode := range g.Nodes() {
		svcID := svcNode.Ext.ID()
		if !strings.HasPrefix(svcID, "service:") {
			continue
		}
		if autoEdgeDisabled(g, svcID) {
			continue
		}
		svcName := strings.TrimPrefix(svcID, "service:")
		if len(svcName) < 3 {
			continue // skip short names to avoid false positives
		}
		for _, fileNode := range g.Nodes() {
			fileID := fileNode.Ext.ID()
			if !strings.HasPrefix(fileID, "file:") {
				continue
			}
			if autoEdgeDisabled(g, fileID) {
				continue
			}
			filePath := strings.TrimPrefix(fileID, "file:")
			// Match as a path component (/svcName/) or exact base filename with known config extension.
			base := filepath.Base(filePath)
			if strings.Contains(filePath, "/"+svcName+"/") ||
				base == svcName+".conf" || base == svcName+".cfg" ||
				base == svcName+".yaml" || base == svcName+".yml" {
				tryAddEdge(g, svcID, fileID)
			}
		}
	}
	return nil
}

// tryAddEdge adds an edge if it won't create a cycle.
func tryAddEdge(g *graph.Graph, fromID, toID string) {
	if g.WouldCycle(fromID, toID) {
		deck.Warningf("auto-edge %s -> %s skipped: would create cycle", fromID, toID)
		return
	}
	if err := g.AddEdge(fromID, toID); err != nil {
		deck.Warningf("auto-edge %s -> %s skipped: %v", fromID, toID, err)
	}
}
