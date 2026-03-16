// Package daemon implements the persistent event-driven convergence loop.
// It watches resources for drift via OS-level events (Watcher interface)
// or polling (Poller interface / default interval), and re-converges
// only the affected resources.
package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/engine"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/output"
	"github.com/google/deck"
)

const defaultPollInterval = 30 * time.Second

// Options controls daemon behavior.
type Options struct {
	Timeout         time.Duration // per-resource timeout
	Parallel        int           // max concurrent resources during initial convergence
	Once            bool          // exit after initial convergence
	DefaultPollFreq time.Duration // poll interval for resources without Watcher or Poller
}

// Daemon watches resources for drift and re-converges them.
type Daemon struct {
	graph   *graph.Graph
	printer output.Printer
	opts    Options
}

// New creates a daemon for the given resource graph.
func New(g *graph.Graph, printer output.Printer, opts Options) *Daemon {
	if opts.DefaultPollFreq == 0 {
		opts.DefaultPollFreq = defaultPollInterval
	}
	return &Daemon{graph: g, printer: printer, opts: opts}
}

// Run performs initial convergence, then watches all resources until ctx
// is cancelled. In Once mode, it returns after initial convergence.
func (d *Daemon) Run(ctx context.Context) error {
	// Phase 1: initial convergence pass.
	engineOpts := engine.Options{
		Timeout:  d.opts.Timeout,
		Parallel: d.opts.Parallel,
	}
	code, err := engine.RunApplyDAG(d.graph, d.printer, engineOpts)
	if err != nil {
		deck.Errorf("initial convergence failed (exit %d): %v", code, err)
	}

	if d.opts.Once {
		return err
	}

	// Phase 2: start watchers/pollers for each resource.
	eventCh := make(chan extensions.Event, 64)
	var wg sync.WaitGroup

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	for _, node := range d.graph.Nodes() {
		ext := node.Ext
		wg.Add(1)

		if w, ok := ext.(extensions.Watcher); ok {
			go func(w extensions.Watcher, ext extensions.Extension) {
				defer wg.Done()
				if err := w.Watch(watchCtx, eventCh); err != nil && watchCtx.Err() == nil {
					deck.Errorf("watcher %s exited: %v", ext.ID(), err)
				}
			}(w, ext)
		} else {
			interval := d.opts.DefaultPollFreq
			if p, ok := ext.(extensions.Poller); ok {
				interval = p.PollInterval()
			}
			go func(ext extensions.Extension, interval time.Duration) {
				defer wg.Done()
				d.poll(watchCtx, ext, interval, eventCh)
			}(ext, interval)
		}
	}

	// Phase 3: event loop, process events until context cancelled.
	d.eventLoop(ctx, eventCh)

	// Shutdown: cancel watchers and wait.
	watchCancel()
	wg.Wait()
	return nil
}

// poll periodically checks a resource and sends an event if it drifts.
func (d *Daemon) poll(ctx context.Context, ext extensions.Extension, interval time.Duration, events chan<- extensions.Event) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, d.opts.Timeout)
			state, err := ext.Check(checkCtx)
			cancel()

			if err != nil {
				deck.Warningf("poll check %s: %v", ext.ID(), err)
				continue
			}
			if state != nil && !state.InSync {
				select {
				case events <- extensions.Event{
					ResourceID: ext.ID(),
					Reason:     "poll detected drift",
					Time:       time.Now(),
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// eventLoop processes incoming events by re-converging the affected resource.
func (d *Daemon) eventLoop(ctx context.Context, events <-chan extensions.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-events:
			node := d.graph.Node(evt.ResourceID)
			if node == nil {
				deck.Warningf("event for unknown resource: %s", evt.ResourceID)
				continue
			}

			deck.Infof("drift detected: %s (%s)", evt.ResourceID, evt.Reason)

			applyCtx, cancel := context.WithTimeout(ctx, d.opts.Timeout)
			state, err := node.Ext.Check(applyCtx)
			if err != nil {
				deck.Errorf("check %s: %v", evt.ResourceID, err)
				cancel()
				continue
			}
			if state == nil || state.InSync {
				cancel()
				continue
			}

			result, err := node.Ext.Apply(applyCtx)
			cancel()
			if err != nil {
				deck.Errorf("apply %s: %v", evt.ResourceID, err)
				continue
			}
			if result != nil {
				d.printer.ApplyResult(node.Ext, result)
			}
		}
	}
}
