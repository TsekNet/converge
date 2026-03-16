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

// Daemon watches resources for drift and re-converges them.
type Daemon struct {
	graph   *graph.Graph
	printer output.Printer
	opts    Options
	states  map[string]*resourceState
	mu      sync.RWMutex
}

// New creates a daemon for the given resource graph.
func New(g *graph.Graph, printer output.Printer, opts Options) *Daemon {
	if opts.DefaultPollFreq == 0 {
		opts.DefaultPollFreq = defaultPollInterval
	}
	if opts.MaxRetries == 0 {
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

	// Phase 3: event loop with retry/backoff.
	d.eventLoop(ctx, eventCh)

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

// eventLoop processes incoming events with retry/backoff logic.
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
			rs.mu.Lock()

			// During backoff, only process scheduled retries and external Watch events
			// (not poll events, which would bypass the backoff window).
			if rs.compliance == Converging && evt.Reason == "poll detected drift" {
				rs.mu.Unlock()
				continue
			}
			if !rs.nextRetry.IsZero() && time.Now().Before(rs.nextRetry) && evt.Reason != "retry" {
				rs.mu.Unlock()
				continue
			}

			// If noncompliant and this is a new external Watch event (not poll/retry),
			// reset retries to give the resource another chance.
			if rs.compliance == Noncompliant && evt.Reason != "retry" && evt.Reason != "poll detected drift" {
				rs.retryCount = 0
				rs.compliance = Converging
				deck.Infof("resetting retries for %s (new external event)", evt.ResourceID)
			}

			rs.mu.Unlock()

			deck.Infof("drift detected: %s (%s)", evt.ResourceID, evt.Reason)
			d.convergeResource(ctx, node.Ext, rs, events)
		}
	}
}

// convergeResource runs Check/Apply with retry/backoff logic.
func (d *Daemon) convergeResource(ctx context.Context, ext extensions.Extension, rs *resourceState, events chan<- extensions.Event) {
	applyCtx, cancel := context.WithTimeout(ctx, d.opts.Timeout)
	defer cancel()

	state, err := ext.Check(applyCtx)
	if err != nil {
		d.handleFailure(ext, rs, err, events)
		return
	}
	if state == nil || state.InSync {
		rs.mu.Lock()
		rs.retryCount = 0
		rs.compliance = Compliant
		rs.lastError = nil
		rs.nextRetry = time.Time{}
		rs.mu.Unlock()
		return
	}

	result, err := ext.Apply(applyCtx)
	if err != nil {
		d.handleFailure(ext, rs, err, events)
		return
	}

	if result != nil {
		d.printer.ApplyResult(ext, result)
	}

	// Success: reset retry state.
	rs.mu.Lock()
	rs.retryCount = 0
	rs.compliance = Compliant
	rs.lastError = nil
	rs.nextRetry = time.Time{}
	rs.mu.Unlock()
}

// handleFailure increments retry count with exponential backoff.
func (d *Daemon) handleFailure(ext extensions.Extension, rs *resourceState, err error, events chan<- extensions.Event) {
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
	// Exponential backoff: baseDelay * 2^(retryCount-1), capped at maxRetryDelay.
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

	// Schedule a retry event after the delay.
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			select {
			case events <- extensions.Event{
				ResourceID: ext.ID(),
				Reason:     "retry",
				Time:       time.Now(),
			}:
			default:
			}
		}
	}()
}
