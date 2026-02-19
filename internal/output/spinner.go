package output

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []string{"◐", "◓", "◑", "◒"}

type Spinner struct {
	mu      sync.Mutex
	active  bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	message string
}

func NewSpinner() *Spinner {
	return &Spinner{}
}

func (s *Spinner) Start(message string) {
	if !isTTY() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		s.stopInternal()
	}

	s.message = message
	s.active = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})

	go func() {
		defer close(s.doneCh)
		i := 0
		for {
			select {
			case <-s.stopCh:
				return
			default:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				fmt.Printf("\r\033[K    %s%s%s %s", colorCyan, spinnerFrames[i%len(spinnerFrames)], colorReset, msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	s.active = false
	doneCh := s.doneCh
	s.mu.Unlock()

	<-doneCh
	fmt.Print("\r\033[K")
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func (s *Spinner) stopInternal() {
	if s.active {
		close(s.stopCh)
		s.active = false
		<-s.doneCh
	}
}
