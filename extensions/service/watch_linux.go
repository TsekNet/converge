//go:build linux

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/TsekNet/converge/extensions"
	"github.com/godbus/dbus/v5"
)

// Watch uses D-Bus to subscribe to systemd PropertiesChanged signals for
// the service unit. When the unit's ActiveState changes, an event is sent.
func (s *Service) Watch(ctx context.Context, events chan<- extensions.Event) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("connect system bus: %w", err)
	}
	defer conn.Close()

	unitName := s.Name + ".service"
	objectPath := dbus.ObjectPath("/org/freedesktop/systemd1/unit/" + escapeUnitName(unitName))

	// Subscribe to PropertiesChanged signals for this unit.
	matchRule := fmt.Sprintf(
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='%s'",
		objectPath,
	)
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule)

	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)

	for {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-signals:
			if sig == nil {
				continue
			}
			select {
			case events <- extensions.Event{
				ResourceID: s.ID(),
				Reason:     "dbus PropertiesChanged",
				Time:       time.Now(),
			}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// escapeUnitName converts a systemd unit name to a D-Bus object path component.
// Characters other than [a-zA-Z0-9] are escaped as _XX (hex).
func escapeUnitName(name string) string {
	var out []byte
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '_')
			out = append(out, "0123456789abcdef"[c>>4])
			out = append(out, "0123456789abcdef"[c&0x0f])
		}
	}
	return string(out)
}
