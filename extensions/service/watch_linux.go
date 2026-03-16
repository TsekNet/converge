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
// the service unit. When the unit's ActiveState or SubState changes, an
// event is sent.
func (s *Service) Watch(ctx context.Context, events chan<- extensions.Event) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("connect system bus: %w", err)
	}
	defer conn.Close()

	unitName := s.Name + ".service"
	objectPath := dbus.ObjectPath("/org/freedesktop/systemd1/unit/" + escapeUnitName(unitName))

	matchRule := fmt.Sprintf(
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='%s'",
		objectPath,
	)
	call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule)
	if call.Err != nil {
		return fmt.Errorf("AddMatch for %s: %w", unitName, call.Err)
	}

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
			// Filter: only emit events for state-related property changes.
			if len(sig.Body) >= 2 {
				iface, ok := sig.Body[0].(string)
				if ok && iface != "org.freedesktop.systemd1.Unit" {
					continue
				}
				if changed, ok := sig.Body[1].(map[string]dbus.Variant); ok {
					_, hasActive := changed["ActiveState"]
					_, hasSub := changed["SubState"]
					if !hasActive && !hasSub {
						continue
					}
				}
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
