package extensions

import "context"

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
