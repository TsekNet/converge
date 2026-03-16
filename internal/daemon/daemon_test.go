package daemon

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/graph"
	"github.com/TsekNet/converge/internal/output"
)

// mockExt implements Extension with configurable Check/Apply behavior.
// All fields accessed from multiple goroutines use atomics.
type mockExt struct {
	id      string
	inSync  atomic.Bool
	applied atomic.Int32
}

func newMockExt(id string, inSync bool) *mockExt {
	m := &mockExt{id: id}
	m.inSync.Store(inSync)
	return m
}

func (m *mockExt) ID() string { return m.id }
func (m *mockExt) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: m.inSync.Load()}, nil
}
func (m *mockExt) Apply(_ context.Context) (*extensions.Result, error) {
	m.applied.Add(1)
	m.inSync.Store(true)
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
	ext := newMockExt("file:/etc/test", false)

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:          5 * time.Second,
		Parallel:         1,
		ConvergedTimeout: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

func TestDaemon_WatcherTriggersApply(t *testing.T) {
	ext := &mockWatcherExt{
		mockExt: *newMockExt("file:/etc/test", true),
		watchFn: func(ctx context.Context, events chan<- extensions.Event) error {
			select {
			case <-time.After(50 * time.Millisecond):
				events <- extensions.Event{
					ResourceID: "file:/etc/test",
					Kind:       extensions.EventWatch,
					Detail:     "mock watcher",
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(40 * time.Millisecond)
		ext.inSync.Store(false)
	}()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

func TestDaemon_PollerFallback(t *testing.T) {
	ext := &mockPollerExt{
		mockExt:  *newMockExt("package:git", true),
		interval: 50 * time.Millisecond,
	}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:  5 * time.Second,
		Parallel: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(75 * time.Millisecond)
		ext.inSync.Store(false)
	}()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

func TestDaemon_DefaultPollInterval(t *testing.T) {
	ext := newMockExt("exec:check", true)

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:         5 * time.Second,
		Parallel:        1,
		DefaultPollFreq: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(75 * time.Millisecond)
		ext.inSync.Store(false)
	}()

	d.Run(ctx)

	if ext.applied.Load() < 1 {
		t.Errorf("Apply called %d times, want >= 1", ext.applied.Load())
	}
}

// mockFailExt always fails Apply.
type mockFailExt struct {
	id         string
	applyCount atomic.Int32
}

func (m *mockFailExt) ID() string { return m.id }
func (m *mockFailExt) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: false}, nil
}
func (m *mockFailExt) Apply(_ context.Context) (*extensions.Result, error) {
	m.applyCount.Add(1)
	return nil, fmt.Errorf("always fails")
}
func (m *mockFailExt) String() string { return m.id }

func TestDaemon_RetryBackoff(t *testing.T) {
	ext := &mockFailExt{id: "file:/etc/broken"}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:         5 * time.Second,
		Parallel:        1,
		MaxRetries:      3,
		RetryBaseDelay:  5 * time.Millisecond,
		DefaultPollFreq: 10 * time.Millisecond,
		CoalesceWindow:  5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Run(ctx)

	status := d.Status(ext.ID())
	if status.Compliance != Noncompliant {
		t.Errorf("compliance = %v, want Noncompliant", status.Compliance)
	}
	if status.RetryCount < 3 {
		t.Errorf("retryCount = %d, want >= 3", status.RetryCount)
	}
}

// mockTransientFailExt fails N times then succeeds, implements Watcher.
type mockTransientFailExt struct {
	id        string
	failUntil int32
	callCount atomic.Int32
	inSync    atomic.Bool
}

func (m *mockTransientFailExt) ID() string { return m.id }
func (m *mockTransientFailExt) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: m.inSync.Load()}, nil
}
func (m *mockTransientFailExt) Apply(_ context.Context) (*extensions.Result, error) {
	n := m.callCount.Add(1)
	if n <= m.failUntil {
		return nil, fmt.Errorf("transient failure %d", n)
	}
	m.inSync.Store(true)
	return &extensions.Result{Status: extensions.StatusChanged, Changed: true}, nil
}
func (m *mockTransientFailExt) String() string { return m.id }
func (m *mockTransientFailExt) Watch(ctx context.Context, events chan<- extensions.Event) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			events <- extensions.Event{
				ResourceID: m.id,
				Kind:       extensions.EventWatch,
				Detail:     "external change",
				Time:       time.Now(),
			}
		}
	}
}

func TestDaemon_RetryResetsOnSuccess(t *testing.T) {
	ext := &mockTransientFailExt{
		id:        "file:/etc/test",
		failUntil: 2,
	}
	// inSync defaults to false (atomic.Bool zero value)

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:        5 * time.Second,
		Parallel:       1,
		MaxRetries:     5,
		RetryBaseDelay: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	d.Run(ctx)

	status := d.Status(ext.ID())
	if status.Compliance == Noncompliant {
		t.Error("should not be noncompliant after successful apply")
	}
}

func TestDaemon_TimeoutExitsAfterStability(t *testing.T) {
	ext := &mockWatcherExt{
		mockExt: *newMockExt("file:/etc/test", true),
		watchFn: func(ctx context.Context, _ chan<- extensions.Event) error {
			<-ctx.Done()
			return nil
		},
	}

	g := graph.New()
	g.AddNode(ext)

	d := New(g, &nullPrinter{}, Options{
		Timeout:          5 * time.Second,
		Parallel:         1,
		ConvergedTimeout: 1 * time.Second,
	})

	start := time.Now()
	d.Run(context.Background())
	elapsed := time.Since(start)

	// Should exit after ~1s of stability, not block forever.
	if elapsed > 5*time.Second {
		t.Errorf("timeout mode took %v, expected ~1s", elapsed)
	}
}
