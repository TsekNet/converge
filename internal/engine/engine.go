package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/output"
	"github.com/google/deck"
)

// Options controls engine execution behaviour.
type Options struct {
	Timeout  time.Duration // per-resource timeout (0 = no timeout)
	Parallel int           // max concurrent resources (<=1 = sequential)
}

func DefaultOptions() Options {
	return Options{Timeout: 5 * time.Minute, Parallel: 1}
}

// CheckDuplicates detects resources with the same ID declared in a blueprint.
func CheckDuplicates(resources []extensions.Extension) error {
	seen := make(map[string]bool, len(resources))
	for _, r := range resources {
		if seen[r.ID()] {
			return fmt.Errorf("duplicate resource: %s", r.ID())
		}
		seen[r.ID()] = true
	}
	return nil
}

func withTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d > 0 {
		return context.WithTimeout(parent, d)
	}
	return parent, func() {}
}

func isCritical(r extensions.Extension) bool {
	cr, ok := r.(extensions.CriticalResource)
	return ok && cr.IsCritical()
}

// RunPlan checks all resources without applying changes.
func RunPlan(resources []extensions.Extension, printer output.Printer, opts Options) (int, error) {
	ctx := context.Background()
	pending, ok := 0, 0

	for i, r := range resources {
		printer.ResourceChecking(r, i+1, len(resources))

		rctx, cancel := withTimeout(ctx, opts.Timeout)
		state, err := r.Check(rctx)
		cancel()

		if err != nil {
			printer.Error(r, err)
			return 1, fmt.Errorf("check failed for %s: %w", r.ID(), err)
		}
		if state == nil {
			state = &extensions.State{}
		}

		printer.PlanResult(r, state)
		if state.InSync {
			ok++
		} else {
			pending++
		}
	}

	printer.PlanSummary(pending, ok, len(resources))
	if pending > 0 {
		return 5, nil
	}
	return 0, nil
}

type applyResult struct {
	ext    extensions.Extension
	result *extensions.Result
}

func (ar applyResult) failed() bool {
	return ar.result.Status == extensions.StatusFailed
}

func applyOne(ctx context.Context, r extensions.Extension, timeout time.Duration) applyResult {
	start := time.Now()

	rctx, cancel := withTimeout(ctx, timeout)
	state, err := r.Check(rctx)
	cancel()

	if err != nil {
		return applyResult{r, &extensions.Result{
			Status: extensions.StatusFailed, Err: err, Duration: time.Since(start),
		}}
	}
	if state == nil {
		state = &extensions.State{}
	}
	if state.InSync {
		return applyResult{r, &extensions.Result{Status: extensions.StatusOK}}
	}

	rctx, cancel = withTimeout(ctx, timeout)
	result, err := r.Apply(rctx)
	cancel()

	if err != nil {
		return applyResult{r, &extensions.Result{
			Status: extensions.StatusFailed, Err: err, Duration: time.Since(start),
		}}
	}
	if result == nil {
		return applyResult{r, &extensions.Result{
			Status: extensions.StatusFailed, Err: fmt.Errorf("Apply returned nil"), Duration: time.Since(start),
		}}
	}
	result.Duration = time.Since(start)
	return applyResult{r, result}
}

// RunApply checks and applies changes for all resources.
func RunApply(resources []extensions.Extension, printer output.Printer, opts Options) (int, error) {
	ctx := context.Background()
	start := time.Now()
	changed, ok, failed := 0, 0, 0

	setMaxNameLen(resources, printer)

	results := make([]applyResult, len(resources))

	if opts.Parallel > 1 {
		sem := make(chan struct{}, opts.Parallel)
		var wg sync.WaitGroup
		for i, r := range resources {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int, res extensions.Extension) {
				defer wg.Done()
				defer func() { <-sem }()
				results[idx] = applyOne(ctx, res, opts.Timeout)
			}(i, r)
		}
		wg.Wait()
	} else {
		for i, r := range resources {
			results[i] = applyOne(ctx, r, opts.Timeout)
		}
	}

	for i, ar := range results {
		printer.ApplyStart(ar.ext, i+1, len(resources))
		printer.ApplyResult(ar.ext, ar.result)

		switch ar.result.Status {
		case extensions.StatusOK:
			ok++
		case extensions.StatusChanged:
			changed++
		default:
			failed++
			if isCritical(ar.ext) {
				deck.Errorf("critical resource failed: %s", ar.ext.ID())
				printer.Summary(changed, ok, failed, changed+ok+failed, time.Since(start).Milliseconds())
				return 3, fmt.Errorf("critical resource %s failed", ar.ext.ID())
			}
		}
	}

	total := changed + ok + failed
	printer.Summary(changed, ok, failed, total, time.Since(start).Milliseconds())

	switch {
	case total == 0:
		return 0, nil
	case failed == total:
		return 4, nil
	case failed > 0:
		return 3, nil
	case changed > 0:
		return 2, nil
	default:
		return 0, nil
	}
}

type nameAware interface {
	SetMaxNameLen(int)
}

func setMaxNameLen(resources []extensions.Extension, printer output.Printer) {
	if p, ok := printer.(nameAware); ok {
		maxLen := 0
		for _, r := range resources {
			if l := len(r.String()); l > maxLen {
				maxLen = l
			}
		}
		p.SetMaxNameLen(maxLen)
	}
}

// RunPlanDAG checks all resources in topological order without applying changes.
func RunPlanDAG(g *graph.Graph, printer output.Printer, opts Options) (int, error) {
	layers, err := g.TopologicalLayers()
	if err != nil {
		return 1, fmt.Errorf("building execution order: %w", err)
	}

	// Flatten for total count and name alignment.
	var all []extensions.Extension
	for _, layer := range layers {
		all = append(all, layer...)
	}

	return RunPlan(all, printer, opts)
}

// RunApplyDAG checks and applies changes in topological layer order.
// Resources within the same layer run concurrently up to opts.Parallel.
// Dependencies in earlier layers complete before later layers start.
func RunApplyDAG(g *graph.Graph, printer output.Printer, opts Options) (int, error) {
	layers, err := g.TopologicalLayers()
	if err != nil {
		return 1, fmt.Errorf("building execution order: %w", err)
	}

	// Flatten for total count and name alignment.
	var all []extensions.Extension
	for _, layer := range layers {
		all = append(all, layer...)
	}

	ctx := context.Background()
	start := time.Now()
	changed, ok, failed := 0, 0, 0
	idx := 0

	setMaxNameLen(all, printer)

	for _, layer := range layers {
		results := make([]applyResult, len(layer))

		if opts.Parallel > 1 && len(layer) > 1 {
			sem := make(chan struct{}, opts.Parallel)
			var wg sync.WaitGroup
			for i, r := range layer {
				wg.Add(1)
				sem <- struct{}{}
				go func(j int, res extensions.Extension) {
					defer wg.Done()
					defer func() { <-sem }()
					results[j] = applyOne(ctx, res, opts.Timeout)
				}(i, r)
			}
			wg.Wait()
		} else {
			for i, r := range layer {
				results[i] = applyOne(ctx, r, opts.Timeout)
			}
		}

		for _, ar := range results {
			idx++
			printer.ApplyStart(ar.ext, idx, len(all))
			printer.ApplyResult(ar.ext, ar.result)

			switch ar.result.Status {
			case extensions.StatusOK:
				ok++
			case extensions.StatusChanged:
				changed++
			default:
				failed++
				if isCritical(ar.ext) {
					deck.Errorf("critical resource failed: %s", ar.ext.ID())
					printer.Summary(changed, ok, failed, changed+ok+failed, time.Since(start).Milliseconds())
					return 3, fmt.Errorf("critical resource %s failed", ar.ext.ID())
				}
			}
		}
	}

	total := changed + ok + failed
	printer.Summary(changed, ok, failed, total, time.Since(start).Milliseconds())

	switch {
	case total == 0:
		return 0, nil
	case failed == total:
		return 4, nil
	case failed > 0:
		return 3, nil
	case changed > 0:
		return 2, nil
	default:
		return 0, nil
	}
}
