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

type Change struct {
	Property string
	From     string
	To       string
	Action   string // "add", "modify", "remove"
}

type State struct {
	InSync  bool
	Changes []Change
}

type Result struct {
	Changed  bool
	Status   Status
	Message  string
	Duration time.Duration
	Err      error
}
