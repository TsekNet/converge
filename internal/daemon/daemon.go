// Package daemon implements the persistent event-driven convergence loop.
// It watches resources for drift via OS-level events (Watcher interface)
// or polling (Poller interface / default interval), and re-converges
// only the affected resources.
package daemon

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/engine"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/output"
	"github.com/google/deck"
)

const (
	defaultPollInterval    = 30 * time.Second
	defaultMaxRetries      = 3
	defaultRetryBase       = 5 * time.Second
	defaultCoalesceWindow  = 500 * time.Millisecond
	defaultRatePerResource = 0.1 // 1 event per 10 seconds
	defaultRateBurst       = 3
)

// Options controls daemon behavior.
type Options struct {
	Timeout          time.Duration // per-resource timeout
	Parallel         int           // max concurrent resources during initial convergence
	Once             bool          // exit after initial convergence
	DefaultPollFreq  time.Duration // poll interval for resources without Watcher or Poller
	MaxRetries       int           // max retries before marking noncompliant (0 = use default)
	RetryBaseDelay   time.Duration // base delay for exponential backoff (0 = use default)
	CoalesceWindow   time.Duration // event coalescing window (0 = use default)
	ConvergedTimeout time.Duration // exit after system is stable for this duration (0 = run forever)
}

// Daemon watches resources for drift and re-converges them.
type Daemon struct {
	graph      *graph.Graph
	printer    output.Printer
	opts       Options
	retries    *retryManager
	initErr    error      // error from initial convergence
	processing sync.Map   // tracks in-progress resource IDs
	lastChange atomic.Int64 // unix nano timestamp of last Apply that changed something
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
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}

	rm := newRetryManager(opts.MaxRetries, opts.RetryBaseDelay)
	for _, node := range g.Nodes() {
		rm.register(node.Ext.ID())
		if node.Meta.Retry > 0 {
			rm.setRetryOverride(node.Ext.ID(), node.Meta.Retry)
		}
	}

	return &Daemon{graph: g, printer: printer, opts: opts, retries: rm}
}

// Status returns the current compliance state of a resource.
func (d *Daemon) Status(id string) ResourceStatus {
	return d.retries.status(id)
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

	// Phase 2: start watchers/pollers feeding raw events.
	rawEvents := make(chan extensions.Event, 256)
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	wg := d.startWatchers(watchCtx, rawEvents)

	// Phase 3: split events into coalesced (watch/poll) and direct (retry).
	coalescedIDs := make(chan string, 256)
	retryIDs := make(chan string, 64)
	window := d.opts.CoalesceWindow
	if window == 0 {
		window = defaultCoalesceWindow
	}
	coalescer := newCoalescer(window, coalescedIDs)
	go coalescer.run(watchCtx)

	// Bridge: raw events -> coalescer or direct retry channel.
	eventMeta := &sync.Map{} // resourceID -> last extensions.EventKind
	go func() {
		for {
			select {
			case <-watchCtx.Done():
				return
			case evt := <-rawEvents:
				eventMeta.Store(evt.ResourceID, evt.Kind)
				if evt.Kind == extensions.EventRetry {
					select {
					case retryIDs <- evt.ResourceID:
					default:
					}
				} else {
					coalescer.submit(evt)
				}
			}
		}
	}()

	// Converged timeout: exit after system is stable for the specified duration.
	loopCtx := ctx
	var convergedCancel context.CancelFunc
	if d.opts.ConvergedTimeout > 0 {
		loopCtx, convergedCancel = context.WithCancel(ctx)
		d.lastChange.Store(time.Now().UnixNano())
		go d.watchConvergence(loopCtx, convergedCancel)
	}

	// Phase 4: rate-limited event loop reads from both channels.
	rl := newResourceRateLimiter(defaultRatePerResource, defaultRateBurst)
	d.processLoop(loopCtx, coalescedIDs, retryIDs, eventMeta, rl, rawEvents)

	if convergedCancel != nil {
		convergedCancel()
	}
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
				backoff := time.Second
				maxBackoff := 5 * time.Minute
				for {
					err := w.Watch(ctx, eventCh)
					if ctx.Err() != nil {
						return
					}
					if err == nil {
						return
					}
					deck.Warningf("watcher %s failed, restarting in %v: %v", ext.ID(), backoff, err)
					select {
					case <-time.After(backoff):
						backoff *= 2
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
					case <-ctx.Done():
						return
					}
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
func (d *Daemon) poll(ctx context.Context, ext extensions.Extension, interval time.Duration, events chan<- extensions.Event) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if d.retries.isNoncompliant(ext.ID()) {
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
					Kind:       extensions.EventPoll,
					Detail:     "poll detected drift",
					Time:       time.Now(),
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// processLoop reads coalesced and retry resource IDs, applies rate limiting,
// and converges each resource in its own goroutine.
func (d *Daemon) processLoop(ctx context.Context, coalescedIDs, retryIDs <-chan string, eventMeta *sync.Map, rl *resourceRateLimiter, rawEvents chan extensions.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-coalescedIDs:
			d.handleResourceEvent(ctx, id, eventMeta, rl, rawEvents)
		case id := <-retryIDs:
			d.handleResourceEvent(ctx, id, eventMeta, rl, rawEvents)
		}
	}
}

func (d *Daemon) handleResourceEvent(ctx context.Context, id string, eventMeta *sync.Map, rl *resourceRateLimiter, rawEvents chan extensions.Event) {
	node := d.graph.Node(id)
	if node == nil {
		return
	}

	kind := extensions.EventWatch
	if v, ok := eventMeta.Load(id); ok {
		kind = v.(extensions.EventKind)
	}

	if !d.retries.shouldProcess(id, kind) {
		return
	}

	// Rate limit watch/poll events, not retries (retries have their own backoff).
	if kind != extensions.EventRetry && !rl.allow(ctx, id) {
		return
	}

	if _, loaded := d.processing.LoadOrStore(id, true); loaded {
		return
	}

	deck.Infof("drift detected: %s (%s)", id, kind)
	go func(ext extensions.Extension, id string) {
		defer d.processing.Delete(id)
		d.convergeResource(ctx, ext, id, rawEvents)
	}(node.Ext, id)
}

// convergeResource runs Check/Apply with retry/backoff logic.
func (d *Daemon) convergeResource(ctx context.Context, ext extensions.Extension, id string, events chan<- extensions.Event) {
	applyCtx, cancel := context.WithTimeout(ctx, d.opts.Timeout)
	defer cancel()

	state, err := ext.Check(applyCtx)
	if err != nil {
		d.scheduleRetry(ctx, id, err, events)
		return
	}
	if state == nil || state.InSync {
		d.retries.reset(id)
		return
	}

	result, err := ext.Apply(applyCtx)
	if err != nil {
		d.scheduleRetry(ctx, id, err, events)
		return
	}

	if result != nil && result.Changed {
		d.lastChange.Store(time.Now().UnixNano())
		d.printer.ApplyResult(ext, result)
	} else if result != nil {
		d.printer.ApplyResult(ext, result)
	}
	d.retries.reset(id)

	// Schedule a re-check for dependent resources so they converge
	// after their dependency has been successfully applied.
	for _, childID := range d.graph.Children(id) {
		select {
		case events <- extensions.Event{
			ResourceID: childID,
			Kind:       extensions.EventWatch,
			Detail:     "dependency converged: " + id,
			Time:       time.Now(),
		}:
		default:
		}
	}
}

// watchConvergence monitors the last change timestamp and cancels the context
// once the system has been stable for ConvergedTimeout.
func (d *Daemon) watchConvergence(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			last := time.Unix(0, d.lastChange.Load())
			if time.Since(last) >= d.opts.ConvergedTimeout {
				deck.Infof("system stable for %v, shutting down", d.opts.ConvergedTimeout)
				cancel()
				return
			}
		}
	}
}

// scheduleRetry records a failure and schedules a retry event after backoff.
func (d *Daemon) scheduleRetry(ctx context.Context, id string, err error, events chan<- extensions.Event) {
	delay := d.retries.recordFailure(id, err)
	if delay == 0 {
		return // noncompliant, no more retries
	}

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			select {
			case events <- extensions.Event{
				ResourceID: id,
				Kind:       extensions.EventRetry,
				Detail:     "scheduled retry",
				Time:       time.Now(),
			}:
			default:
			}
		case <-ctx.Done():
			return
		}
	}()
}
