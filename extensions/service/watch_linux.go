//go:build linux

package service

import (
	"context"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/watch"
)

// Watch uses the shared D-Bus multiplexer to subscribe to systemd
// PropertiesChanged signals for the service unit. When the unit's ActiveState
// or SubState changes, an event is sent.
func (s *Service) Watch(ctx context.Context, events chan<- extensions.Event) error {
	w, err := watch.SharedDbus()
	if err != nil {
		return err
	}

	unitName := s.Name + ".service"
	ch, err := w.WatchUnit(unitName)
	if err != nil {
		return err
	}
	defer w.UnwatchUnit(unitName, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ch:
			if !ok {
				return nil
			}
			select {
			case events <- extensions.Event{
				ResourceID: s.ID(),
				Kind:       extensions.EventWatch, Detail: "dbus PropertiesChanged",
				Time: time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
