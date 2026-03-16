package extensions

import "time"

type Status int

const (
	StatusOK Status = iota
	StatusChanged
	StatusFailed
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusChanged:
		return "changed"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Change describes a single property that differs between current and desired state.
type Change struct {
	Property string
	From     string
	To       string
	Action   string // "add", "modify", "remove"
}

// State is returned by Check: InSync means no changes needed.
type State struct {
	InSync  bool
	Changes []Change
}

// Result is returned by Apply: Changed indicates whether the system was modified.
type Result struct {
	Changed  bool
	Status   Status
	Message  string
	Duration time.Duration
	Err      error
	Changes  []Change // drift details from Check (populated by engine for display)
}
