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

// Condition is optionally set on ResourceMeta to gate convergence on system
// state. The daemon skips the resource until Met returns true, then triggers
// initial convergence. Conditions do not affect ongoing drift detection after
// they are first satisfied.
//
// Implementations must use OS-native APIs (netlink, inotify, Win32, etc.).
// Wait must block using kernel events, not a polling loop, unless no kernel
// API exists for the condition type (documented per implementation).
type Condition interface {
	// Met returns true if the condition is currently satisfied.
	Met(ctx context.Context) (bool, error)
	// Wait blocks until the condition becomes true, ctx is cancelled,
	// or a fatal error occurs. Returns nil when the condition is met.
	Wait(ctx context.Context) error
	// String returns a human-readable description for logging.
	String() string
}

// EventKind classifies how an event was generated.
type EventKind int

const (
	EventWatch     EventKind = iota // OS-level watcher detected change
	EventPoll                       // periodic poll detected drift
	EventRetry                      // scheduled retry after failure
	EventCondition                  // condition became true, triggering initial convergence
)

func (k EventKind) String() string {
	switch k {
	case EventWatch:
		return "watch"
	case EventPoll:
		return "poll"
	case EventRetry:
		return "retry"
	case EventCondition:
		return "condition"
	default:
		return "unknown"
	}
}

// Event signals that a resource may need reconciliation.
type Event struct {
	ResourceID string
	Kind       EventKind
	Detail     string // human-readable context (e.g. "inotify", "kqueue")
	Time       time.Time
}
