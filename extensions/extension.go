package extensions

import "context"

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
