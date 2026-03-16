package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/output"
)

// mockExt implements Extension with configurable Check/Apply behavior.
type mockExt struct {
	id      string
	inSync  bool
	applied atomic.Int32
}

func (m *mockExt) ID() string { return m.id }
func (m *mockExt) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: m.inSync}, nil
}
func (m *mockExt) Apply(_ context.Context) (*extensions.Result, error) {
	m.applied.Add(1)
	m.inSync = true
	return &extensions.Result{Status: extensions.StatusChanged, Changed: true}, nil
}
func (m *mockExt) String() string { return m.id }

// mockWatcherExt implements Extension + Watcher.
type mockWatcherExt struct {
	mockExt
	watchFn func(ctx context.Context, events chan<- extensions.Event) error
}

func (m *mockWatcherExt) Watch(ctx context.Context, events chan<- extensions.Event) error {
	return m.watchFn(ctx, events)
}

// mockPollerExt implements Extension + Poller.
type mockPollerExt struct {
	mockExt
	interval time.Duration
}

func (m *mockPollerExt) PollInterval() time.Duration { return m.interval }

type nullPrinter struct{}

func (p *nullPrinter) Banner(_ string)                                          {}
func (p *nullPrinter) BlueprintHeader(_ string)                                 {}
func (p *nullPrinter) ResourceChecking(_ extensions.Extension, _, _ int)        {}
func (p *nullPrinter) PlanResult(_ extensions.Extension, _ *extensions.State)   {}
func (p *nullPrinter) ApplyStart(_ extensions.Extension, _, _ int)              {}
func (p *nullPrinter) ApplyResult(_ extensions.Extension, _ *extensions.Result) {}
func (p *nullPrinter) Summary(_, _, _, _ int, _ int64)                          {}
func (p *nullPrinter) PlanSummary(_, _, _ int)                                  {}
func (p *nullPrinter) Error(_ extensions.Extension, _ error)                    {}

var _ output.Printer = (*nullPrinter)(nil)

func TestDaemon_InitialConvergence(t *testing.T) {
	ext := &mockExt{id: "file:/etc/test", inSync: false}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:  5 * time.Second,
		Parallel: 1,
		Once:     true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if ext.applied.Load() != 1 {
		t.Errorf("Apply called %d times, want 1", ext.applied.Load())
	}
}

func TestDaemon_WatcherTriggersApply(t *testing.T) {
	ext := &mockWatcherExt{
		mockExt: mockExt{id: "file:/etc/test", inSync: true},
		watchFn: func(ctx context.Context, events chan<- extensions.Event) error {
			// Send one event after a short delay, then wait for cancellation.
			select {
			case <-time.After(50 * time.Millisecond):
				events <- extensions.Event{
					ResourceID: "file:/etc/test",
					Reason:     "file modified",
					Time:       time.Now(),
				}
			case <-ctx.Done():
				return nil
			}
			<-ctx.Done()
			return nil
		},
	}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:  5 * time.Second,
		Parallel: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Before the event, set inSync=false so Apply runs when event fires.
	go func() {
		time.Sleep(40 * time.Millisecond)
		ext.inSync = false
	}()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

func TestDaemon_PollerFallback(t *testing.T) {
	ext := &mockPollerExt{
		mockExt:  mockExt{id: "package:git", inSync: true},
		interval: 50 * time.Millisecond,
	}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:  5 * time.Second,
		Parallel: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// After 75ms, mark out of sync so next poll triggers Apply.
	go func() {
		time.Sleep(75 * time.Millisecond)
		ext.inSync = false
	}()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

func TestDaemon_DefaultPollInterval(t *testing.T) {
	// Extension that implements neither Watcher nor Poller uses default poll.
	ext := &mockExt{id: "exec:check", inSync: true}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:         5 * time.Second,
		Parallel:        1,
		DefaultPollFreq: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() {
		time.Sleep(75 * time.Millisecond)
		ext.inSync = false
	}()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

func TestDaemon_OnceExitsAfterConvergence(t *testing.T) {
	ext := &mockWatcherExt{
		mockExt: mockExt{id: "file:/etc/test", inSync: true},
		watchFn: func(ctx context.Context, _ chan<- extensions.Event) error {
			<-ctx.Done()
			return nil
		},
	}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:  5 * time.Second,
		Parallel: 1,
		Once:     true,
	})

	start := time.Now()
	ctx := context.Background()
	d.Run(ctx)
	elapsed := time.Since(start)

	// Once mode should return quickly, not block forever.
	if elapsed > 2*time.Second {
		t.Errorf("Once mode took %v, expected quick return", elapsed)
	}
}
