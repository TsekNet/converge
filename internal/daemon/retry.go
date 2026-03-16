package daemon

import (
	"sync"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/google/deck"
)

// Compliance represents a resource's convergence state.
type Compliance int

const (
	Compliant    Compliance = iota
	Noncompliant            // exceeded max retries
	Converging              // actively retrying
)

// ResourceStatus tracks runtime state for a resource in daemon mode.
type ResourceStatus struct {
	Compliance Compliance
	RetryCount int
	LastError  error
}

// retryManager tracks per-resource retry/backoff/compliance state.
type retryManager struct {
	states     map[string]*resourceState
	mu         sync.RWMutex
	maxRetries int
	baseDelay  time.Duration
}

// resourceState tracks per-resource retry and compliance state.
type resourceState struct {
	mu         sync.Mutex
	retryCount int
	nextRetry  time.Time
	compliance Compliance
	lastError  error
}

func newRetryManager(maxRetries int, baseDelay time.Duration) *retryManager {
	return &retryManager{
		states:     make(map[string]*resourceState),
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

func (rm *retryManager) register(id string) {
	rm.states[id] = &resourceState{}
}

func (rm *retryManager) status(id string) ResourceStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	s, ok := rm.states[id]
	if !ok {
		return ResourceStatus{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return ResourceStatus{
		Compliance: s.compliance,
		RetryCount: s.retryCount,
		LastError:  s.lastError,
	}
}

// shouldProcess determines if an event should trigger convergence.
func (rm *retryManager) shouldProcess(id string, kind extensions.EventKind) bool {
	s := rm.states[id]
	s.mu.Lock()
	defer s.mu.Unlock()

	// During backoff, skip poll events.
	if s.compliance == Converging && kind == extensions.EventPoll {
		return false
	}
	// During backoff window, only process scheduled retries.
	if !s.nextRetry.IsZero() && time.Now().Before(s.nextRetry) && kind != extensions.EventRetry {
		return false
	}
	// Noncompliant resources reset on new external Watch events.
	if s.compliance == Noncompliant && kind == extensions.EventWatch {
		s.retryCount = 0
		s.compliance = Converging
		deck.Infof("resetting retries for %s (external watch event)", id)
	}
	return true
}

// isNoncompliant checks if a resource is marked noncompliant.
func (rm *retryManager) isNoncompliant(id string) bool {
	s := rm.states[id]
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.compliance == Noncompliant
}

// reset marks a resource as compliant with zero retries.
func (rm *retryManager) reset(id string) {
	s := rm.states[id]
	s.mu.Lock()
	s.retryCount = 0
	s.compliance = Compliant
	s.lastError = nil
	s.nextRetry = time.Time{}
	s.mu.Unlock()
}

const maxRetryDelay = 5 * time.Minute

// recordFailure increments retry count with exponential backoff.
// Returns the backoff delay, or 0 if noncompliant (no more retries).
func (rm *retryManager) recordFailure(id string, err error) time.Duration {
	s := rm.states[id]
	s.mu.Lock()
	defer s.mu.Unlock()

	s.retryCount++
	s.lastError = err

	if s.retryCount >= rm.maxRetries {
		s.compliance = Noncompliant
		deck.Warningf("resource %s noncompliant after %d retries: %v", id, s.retryCount, err)
		return 0
	}

	s.compliance = Converging
	delay := rm.baseDelay
	for i := 1; i < s.retryCount; i++ {
		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
			break
		}
	}
	s.nextRetry = time.Now().Add(delay)

	deck.Infof("retry %d/%d for %s in %v: %v", s.retryCount, rm.maxRetries, id, delay, err)
	return delay
}
