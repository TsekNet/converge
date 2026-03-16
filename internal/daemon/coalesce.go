package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/time/rate"
)

// coalescer collapses multiple rapid events for the same resource into
// a single notification after a configurable window.
type coalescer struct {
	window  time.Duration
	out     chan<- string // resource IDs to process
	in      chan extensions.Event
	pending map[string]*time.Timer
	mu      sync.Mutex
}

func newCoalescer(window time.Duration, out chan<- string) *coalescer {
	return &coalescer{
		window:  window,
		out:     out,
		in:      make(chan extensions.Event, 128),
		pending: make(map[string]*time.Timer),
	}
}

// submit queues an event for coalescing.
func (c *coalescer) submit(evt extensions.Event) {
	select {
	case c.in <- evt:
	default:
		// Drop event if queue is full (backpressure).
	}
}

// run processes incoming events and fires coalesced outputs.
func (c *coalescer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			for _, t := range c.pending {
				t.Stop()
			}
			c.mu.Unlock()
			return
		case evt := <-c.in:
			c.mu.Lock()
			if _, ok := c.pending[evt.ResourceID]; ok {
				// Already pending: coalesce (drop this event).
				c.mu.Unlock()
				continue
			}
			id := evt.ResourceID
			c.pending[id] = time.AfterFunc(c.window, func() {
				c.mu.Lock()
				delete(c.pending, id)
				c.mu.Unlock()
				select {
				case c.out <- id:
				default:
				}
			})
			c.mu.Unlock()
		}
	}
}

// resourceRateLimiter provides per-resource rate limiting.
type resourceRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rateVal  rate.Limit
	burst    int
}

func newResourceRateLimiter(r float64, burst int) *resourceRateLimiter {
	return &resourceRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rateVal:  rate.Limit(r),
		burst:    burst,
	}
}

// allow returns true if the event should be processed, false if rate-limited.
func (rl *resourceRateLimiter) allow(ctx context.Context, id string) bool {
	rl.mu.RLock()
	l, ok := rl.limiters[id]
	rl.mu.RUnlock()

	if !ok {
		rl.mu.Lock()
		l, ok = rl.limiters[id]
		if !ok {
			l = rate.NewLimiter(rl.rateVal, rl.burst)
			rl.limiters[id] = l
		}
		rl.mu.Unlock()
	}

	return l.Wait(ctx) == nil
}
