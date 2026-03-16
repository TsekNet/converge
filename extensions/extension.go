package extensions

import (
	"context"
	"time"
)

// Extension is the core interface every resource type implements.
// The engine calls Check to detect drift, then Apply to fix it.
type Extension interface {
	ID() string
	Check(ctx context.Context) (*State, error)
	Apply(ctx context.Context) (*Result, error)
	String() string
}

// CriticalResource is optionally implemented by extensions that can halt a run on failure.
type CriticalResource interface {
	IsCritical() bool
}

// Watcher is optionally implemented by extensions that support OS-level
// event watching. Watch blocks until ctx is cancelled, sending events
// on the channel when the resource may have drifted.
type Watcher interface {
	Watch(ctx context.Context, events chan<- Event) error
}

// Poller is optionally implemented by extensions that lack native OS
// event support. The daemon polls Check at this interval instead.
type Poller interface {
	PollInterval() time.Duration
}

// Event signals that a resource may need reconciliation.
type Event struct {
	ResourceID string
	Reason     string
	Time       time.Time
}
