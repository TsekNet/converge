//go:build unix

package logging

import (
	"github.com/google/deck"
	"github.com/google/deck/backends/syslog"
)

func init() {
	sl, err := syslog.Init(AppID, syslog.LOG_USER)
	if err != nil {
		return
	}
	deck.Add(sl)
}
