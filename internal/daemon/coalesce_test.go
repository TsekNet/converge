package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TsekNet/converge/extensions"
)

func TestCoalescer_CollapsesBurstEvents(t *testing.T) {
	out := make(chan string, 10)
	c := newCoalescer(100*time.Millisecond, out)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.run(ctx)

	// Send 50 events for the same resource in rapid succession.
	for i := 0; i < 50; i++ {
		c.submit(extensions.Event{ResourceID: "file:/etc/test", Reason: "modified"})
	}

	// Wait for the coalesce window to fire.
	time.Sleep(200 * time.Millisecond)
	cancel()

	var count int
	for range out {
		count++
		if len(out) == 0 {
			break
		}
	}

	// All 50 events should collapse into 1.
	if count != 1 {
		t.Errorf("got %d coalesced events, want 1", count)
	}
}

func TestCoalescer_DifferentResourcesNotCoalesced(t *testing.T) {
	out := make(chan string, 10)
	c := newCoalescer(50*time.Millisecond, out)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.run(ctx)

	c.submit(extensions.Event{ResourceID: "file:/etc/a", Reason: "modified"})
	c.submit(extensions.Event{ResourceID: "file:/etc/b", Reason: "modified"})

	time.Sleep(150 * time.Millisecond)
	cancel()

	var ids []string
	for id := range out {
		ids = append(ids, id)
		if len(out) == 0 {
			break
		}
	}

	if len(ids) != 2 {
		t.Errorf("got %d events, want 2 (one per resource)", len(ids))
	}
}

func TestRateLimiter_ThrottlesEvents(t *testing.T) {
	// Rate limit: 2 per second, burst 1.
	rl := newResourceRateLimiter(2, 1)
	id := "file:/etc/test"

	var allowed atomic.Int32
	start := time.Now()

	// Try 10 events in 200ms. With rate=2/s and burst=1, we should get ~1-2.
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		if rl.allow(ctx, id) {
			allowed.Add(1)
		}
		cancel()
	}

	elapsed := time.Since(start)
	got := int(allowed.Load())

	// Should allow 1-3 events in ~200ms at 2/s rate.
	if got < 1 || got > 5 {
		t.Errorf("allowed %d events in %v, expected 1-5 at rate 2/s", got, elapsed)
	}
}
