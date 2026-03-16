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

const (
	defaultPollInterval = 30 * time.Second
	defaultMaxRetries   = 3
	defaultRetryBase    = 5 * time.Second
	maxRetryDelay       = 5 * time.Minute
)

// Event reason constants used for routing logic.
const (
	reasonPoll  = "poll detected drift"
	reasonRetry = "retry"
)

// Compliance represents a resource's convergence state.
type Compliance int

const (
	Compliant    Compliance = iota
	Noncompliant            // exceeded max retries
	Converging              // actively retrying
)

// ResourceStatus tracks runtime state for a resource in daemon mode.
type ResourceStatus struct {
	Compliance Compliance
	RetryCount int
	LastError  error
}

// Options controls daemon behavior.
type Options struct {
	Timeout         time.Duration // per-resource timeout
	Parallel        int           // max concurrent resources during initial convergence
	Once            bool          // exit after initial convergence
	DefaultPollFreq time.Duration // poll interval for resources without Watcher or Poller
	MaxRetries      int           // max retries before marking noncompliant (0 = use default)
	RetryBaseDelay  time.Duration // base delay for exponential backoff (0 = use default)
}

// resourceState tracks per-resource retry and compliance state.
type resourceState struct {
	mu         sync.Mutex
	retryCount int
	nextRetry  time.Time
	compliance Compliance
	lastError  error
}

func (rs *resourceState) reset() {
	rs.mu.Lock()
	rs.retryCount = 0
	rs.compliance = Compliant
	rs.lastError = nil
	rs.nextRetry = time.Time{}
	rs.mu.Unlock()
}

func (rs *resourceState) shouldProcess(reason string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// During backoff, skip poll events.
	if rs.compliance == Converging && reason == reasonPoll {
		return false
	}
	// During backoff window, only process scheduled retries.
	if !rs.nextRetry.IsZero() && time.Now().Before(rs.nextRetry) && reason != reasonRetry {
		return false
	}
	// Noncompliant resources reset on new external Watch events.
	if rs.compliance == Noncompliant && reason != reasonRetry && reason != reasonPoll {
		rs.retryCount = 0
		rs.compliance = Converging
		deck.Infof("resetting retries for external event")
	}
	return true
}

// Daemon watches resources for drift and re-converges them.
type Daemon struct {
	graph      *graph.Graph
	printer    output.Printer
	opts       Options
	states     map[string]*resourceState
	mu         sync.RWMutex
	initErr    error // error from initial convergence
	processing sync.Map // tracks in-progress resource IDs
}

// New creates a daemon for the given resource graph.
func New(g *graph.Graph, printer output.Printer, opts Options) *Daemon {
	if opts.DefaultPollFreq == 0 {
		opts.DefaultPollFreq = defaultPollInterval
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = defaultMaxRetries
	}
	if opts.RetryBaseDelay == 0 {
		opts.RetryBaseDelay = defaultRetryBase
	}

	states := make(map[string]*resourceState)
	for _, node := range g.Nodes() {
		states[node.Ext.ID()] = &resourceState{}
	}

	return &Daemon{graph: g, printer: printer, opts: opts, states: states}
}

// Status returns the current compliance state of a resource.
func (d *Daemon) Status(id string) ResourceStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s, ok := d.states[id]
	if !ok {
		return ResourceStatus{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return ResourceStatus{
		Compliance: s.compliance,
		RetryCount: s.retryCount,
		LastError:  s.lastError,
	}
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
		d.initErr = err
	}

	if d.opts.Once {
		return err
	}

	// Phase 2: start watchers/pollers.
	eventCh := make(chan extensions.Event, 256)
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	wg := d.startWatchers(watchCtx, eventCh)

	// Phase 3: event loop with retry/backoff.
	d.eventLoop(ctx, eventCh)

	watchCancel()
	wg.Wait()
	return d.initErr
}

// startWatchers launches a goroutine per resource for Watch or poll.
func (d *Daemon) startWatchers(ctx context.Context, eventCh chan extensions.Event) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, node := range d.graph.Nodes() {
		ext := node.Ext
		wg.Add(1)

		if w, ok := ext.(extensions.Watcher); ok {
			go func(w extensions.Watcher, ext extensions.Extension) {
				defer wg.Done()
				if err := w.Watch(ctx, eventCh); err != nil && ctx.Err() == nil {
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
				d.poll(ctx, ext, interval, eventCh)
			}(ext, interval)
		}
	}

	return &wg
}

// poll periodically checks a resource and sends an event if it drifts.
// Skips Check for noncompliant resources to avoid wasting cycles.
func (d *Daemon) poll(ctx context.Context, ext extensions.Extension, interval time.Duration, events chan<- extensions.Event) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Skip polling for noncompliant resources.
			rs := d.states[ext.ID()]
			rs.mu.Lock()
			nc := rs.compliance == Noncompliant
			rs.mu.Unlock()
			if nc {
				continue
			}

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
					Reason:     reasonPoll,
					Time:       time.Now(),
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// eventLoop processes incoming events. Each resource is converged in its
// own goroutine, but only one convergence runs per resource at a time.
func (d *Daemon) eventLoop(ctx context.Context, events chan extensions.Event) {
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

			rs := d.states[evt.ResourceID]
			if !rs.shouldProcess(evt.Reason) {
				continue
			}

			// Prevent concurrent convergence of the same resource.
			if _, loaded := d.processing.LoadOrStore(evt.ResourceID, true); loaded {
				continue
			}

			deck.Infof("drift detected: %s (%s)", evt.ResourceID, evt.Reason)
			go func(ext extensions.Extension, rs *resourceState) {
				defer d.processing.Delete(ext.ID())
				d.convergeResource(ctx, ext, rs, events)
			}(node.Ext, rs)
		}
	}
}

// convergeResource runs Check/Apply with retry/backoff logic.
func (d *Daemon) convergeResource(ctx context.Context, ext extensions.Extension, rs *resourceState, events chan<- extensions.Event) {
	applyCtx, cancel := context.WithTimeout(ctx, d.opts.Timeout)
	defer cancel()

	state, err := ext.Check(applyCtx)
	if err != nil {
		d.handleFailure(ctx, ext, rs, err, events)
		return
	}
	if state == nil || state.InSync {
		rs.reset()
		return
	}

	result, err := ext.Apply(applyCtx)
	if err != nil {
		d.handleFailure(ctx, ext, rs, err, events)
		return
	}

	if result != nil {
		d.printer.ApplyResult(ext, result)
	}
	rs.reset()
}

// handleFailure increments retry count with exponential backoff.
// The retry timer goroutine respects ctx for clean shutdown.
func (d *Daemon) handleFailure(ctx context.Context, ext extensions.Extension, rs *resourceState, err error, events chan<- extensions.Event) {
	rs.mu.Lock()
	rs.retryCount++
	rs.lastError = err

	if rs.retryCount >= d.opts.MaxRetries {
		rs.compliance = Noncompliant
		deck.Warningf("resource %s noncompliant after %d retries: %v", ext.ID(), rs.retryCount, err)
		rs.mu.Unlock()
		return
	}

	rs.compliance = Converging
	delay := d.opts.RetryBaseDelay
	for i := 1; i < rs.retryCount; i++ {
		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
			break
		}
	}
	rs.nextRetry = time.Now().Add(delay)
	retryCount := rs.retryCount
	rs.mu.Unlock()

	deck.Infof("retry %d/%d for %s in %v: %v", retryCount, d.opts.MaxRetries, ext.ID(), delay, err)

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			select {
			case events <- extensions.Event{
				ResourceID: ext.ID(),
				Reason:     reasonRetry,
				Time:       time.Now(),
			}:
			default:
			}
		case <-ctx.Done():
			return
		}
	}()
}
