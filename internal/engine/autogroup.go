package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
	extpkg "github.com/TsekNet/converge/extensions/pkg"
	"github.com/TsekNet/converge/internal/graph"
)

// PackageGroup batches multiple Package resources with the same manager and
// state into a single install/remove invocation. Individual packages are
// still checked one-by-one so the result accurately reports which changed.
type PackageGroup struct {
	Packages []*extpkg.Package
	Manager  extpkg.PackageManager
	State    string // "present" or "absent"
}

func (pg *PackageGroup) ID() string {
	names := make([]string, len(pg.Packages))
	for i, p := range pg.Packages {
		names[i] = p.PkgName
	}
	sort.Strings(names)
	return fmt.Sprintf("packages:%s", strings.Join(names, ","))
}

func (pg *PackageGroup) String() string {
	names := make([]string, len(pg.Packages))
	for i, p := range pg.Packages {
		names[i] = p.PkgName
	}
	sort.Strings(names)
	return fmt.Sprintf("Packages [%s]", strings.Join(names, ", "))
}

func (pg *PackageGroup) IsCritical() bool {
	for _, p := range pg.Packages {
		if p.Critical {
			return true
		}
	}
	return false
}

// Check inspects each package individually, returns InSync only if all are.
func (pg *PackageGroup) Check(ctx context.Context) (*extensions.State, error) {
	var allChanges []extensions.Change
	for _, p := range pg.Packages {
		state, err := p.Check(ctx)
		if err != nil {
			return nil, fmt.Errorf("check %s: %w", p.PkgName, err)
		}
		if state != nil && !state.InSync {
			allChanges = append(allChanges, state.Changes...)
		}
	}
	if len(allChanges) == 0 {
		return &extensions.State{InSync: true}, nil
	}
	return &extensions.State{InSync: false, Changes: allChanges}, nil
}

// Apply uses BatchInstaller if available, otherwise falls back to
// individual Install/Remove calls.
func (pg *PackageGroup) Apply(ctx context.Context) (*extensions.Result, error) {
	// Collect names that actually need action (not already in desired state).
	var needed []string
	for _, p := range pg.Packages {
		state, err := p.Check(ctx)
		if err != nil {
			return nil, fmt.Errorf("pre-apply check %s: %w", p.PkgName, err)
		}
		if state != nil && !state.InSync {
			needed = append(needed, p.PkgName)
		}
	}
	if len(needed) == 0 {
		return &extensions.Result{Status: extensions.StatusOK}, nil
	}

	start := time.Now()

	if bi, ok := pg.Manager.(extpkg.BatchInstaller); ok {
		var err error
		if pg.State == "present" {
			err = bi.InstallBatch(ctx, needed)
		} else {
			err = bi.RemoveBatch(ctx, needed)
		}
		if err != nil {
			return nil, err
		}
	} else {
		for _, name := range needed {
			var err error
			if pg.State == "present" {
				err = pg.Manager.Install(ctx, name)
			} else {
				err = pg.Manager.Remove(ctx, name)
			}
			if err != nil {
				return nil, err
			}
		}
	}

	action := "installed"
	if pg.State != "present" {
		action = "removed"
	}
	return &extensions.Result{
		Changed:  true,
		Status:   extensions.StatusChanged,
		Message:  fmt.Sprintf("%s %d packages", action, len(needed)),
		Duration: time.Since(start),
	}, nil
}

// autoGroupLayer scans a topological layer of nodes and replaces
// individual Package extensions that share the same manager and state
// with a single PackageGroup. Packages with AutoGroup=false are excluded.
func autoGroupLayer(layer []*graph.Node) []extensions.Extension {
	type groupKey struct {
		manager string
		state   string
	}

	groups := make(map[groupKey][]*extpkg.Package)
	groupOrder := make(map[groupKey]int)
	var result []extensions.Extension
	orderIdx := 0

	for _, node := range layer {
		p, ok := node.Ext.(*extpkg.Package)
		if !ok {
			result = append(result, node.Ext)
			continue
		}
		// Respect per-resource AutoGroup override.
		if node.Meta.AutoGroup != nil && !*node.Meta.AutoGroup {
			result = append(result, node.Ext)
			continue
		}
		key := groupKey{manager: p.ManagerName, state: p.State}
		if _, exists := groupOrder[key]; !exists {
			groupOrder[key] = orderIdx
			orderIdx++
		}
		groups[key] = append(groups[key], p)
	}

	// Build PackageGroups from groups with >1 member.
	type orderedGroup struct {
		ext   extensions.Extension
		order int
	}
	var grouped []orderedGroup
	for key, pkgs := range groups {
		if len(pkgs) == 1 {
			grouped = append(grouped, orderedGroup{ext: pkgs[0], order: groupOrder[key]})
		} else {
			pg := &PackageGroup{
				Packages: pkgs,
				Manager:  pkgs[0].Manager,
				State:    key.state,
			}
			grouped = append(grouped, orderedGroup{ext: pg, order: groupOrder[key]})
		}
	}

	// Stable sort: non-package resources first (in original order),
	// then grouped packages in their first-seen order.
	sort.Slice(grouped, func(i, j int) bool {
		return grouped[i].order < grouped[j].order
	})

	for _, g := range grouped {
		result = append(result, g.ext)
	}
	return result
}
