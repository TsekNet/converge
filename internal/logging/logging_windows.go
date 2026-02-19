//go:build windows

package logging

import (
	"github.com/google/deck"
	"github.com/google/deck/backends/eventlog"
)

func init() {
	evt, err := eventlog.Init("Converge")
	if err != nil {
		return
	}
	deck.Add(evt)
}
