//go:build linux

// Package watch provides a shared D-Bus connection multiplexer that uses a
// single system bus connection for all systemd unit watchers, avoiding
// per-resource connection overhead.
package watch

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

// dbusSubscriber holds the notification channel for one caller.
type dbusSubscriber struct {
	ch chan struct{}
}

// DbusWatcher multiplexes many unit watches onto a single D-Bus system bus
// connection. Safe for concurrent use.
type DbusWatcher struct {
	mu   sync.Mutex
	conn *dbus.Conn

	// objectPath -> list of subscribers.
	pathSubs map[dbus.ObjectPath][]*dbusSubscriber
	// unitName -> objectPath, for removal.
	unitToPath map[string]dbus.ObjectPath

	running bool
	done    chan struct{}
}

var (
	globalDbusWatcher *DbusWatcher
	globalDbusOnce    sync.Once
	globalDbusErr     error
)

// SharedDbus returns the process-wide shared DbusWatcher, creating it on first
// call. Returns an error only if the D-Bus connection fails.
func SharedDbus() (*DbusWatcher, error) {
	globalDbusOnce.Do(func() {
		w, err := newDbusWatcher()
		if err != nil {
			globalDbusErr = err
			return
		}
		globalDbusWatcher = w
	})
	return globalDbusWatcher, globalDbusErr
}

func newDbusWatcher() (*DbusWatcher, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system bus: %w", err)
	}

	return &DbusWatcher{
		conn:       conn,
		pathSubs:   make(map[dbus.ObjectPath][]*dbusSubscriber),
		unitToPath: make(map[string]dbus.ObjectPath),
		done:       make(chan struct{}),
	}, nil
}

// WatchUnit subscribes to PropertiesChanged signals for the given systemd unit
// name (e.g. "sshd.service"). Returns a channel that receives a struct{}{} each
// time ActiveState or SubState changes. The returned channel is buffered
// (capacity 1) so a slow consumer does not block other subscribers.
func (w *DbusWatcher) WatchUnit(unitName string) (<-chan struct{}, error) {
	objectPath := unitObjectPath(unitName)

	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan struct{}, 1)
	sub := &dbusSubscriber{ch: ch}

	_, exists := w.unitToPath[unitName]
	if !exists {
		// Add D-Bus match rule for this unit's object path.
		matchRule := fmt.Sprintf(
			"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='%s'",
			objectPath,
		)
		call := w.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule)
		if call.Err != nil {
			return nil, fmt.Errorf("AddMatch for %s: %w", unitName, call.Err)
		}
		w.unitToPath[unitName] = objectPath
	}

	w.pathSubs[objectPath] = append(w.pathSubs[objectPath], sub)

	if !w.running {
		w.running = true
		signals := make(chan *dbus.Signal, 64)
		w.conn.Signal(signals)
		go w.readLoop(signals)
	}

	return ch, nil
}

// UnwatchUnit removes one subscriber channel for a unit name. When the last
// subscriber for a unit is removed, the D-Bus match rule is also removed.
func (w *DbusWatcher) UnwatchUnit(unitName string, ch <-chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	objectPath, ok := w.unitToPath[unitName]
	if !ok {
		return
	}

	subs := w.pathSubs[objectPath]
	for i, s := range subs {
		if s.ch == ch {
			subs[i] = subs[len(subs)-1]
			subs = subs[:len(subs)-1]
			close(s.ch)
			break
		}
	}

	if len(subs) == 0 {
		delete(w.pathSubs, objectPath)
		delete(w.unitToPath, unitName)
		// Best-effort removal of the match rule.
		matchRule := fmt.Sprintf(
			"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='%s'",
			objectPath,
		)
		w.conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, matchRule)
	} else {
		w.pathSubs[objectPath] = subs
	}
}

// Close shuts down the watcher, closing the D-Bus connection.
// All subscriber channels are closed.
func (w *DbusWatcher) Close() error {
	w.mu.Lock()
	for _, subs := range w.pathSubs {
		for _, s := range subs {
			close(s.ch)
		}
	}
	w.pathSubs = make(map[dbus.ObjectPath][]*dbusSubscriber)
	w.unitToPath = make(map[string]dbus.ObjectPath)
	w.mu.Unlock()

	return w.conn.Close()
}

func (w *DbusWatcher) readLoop(signals <-chan *dbus.Signal) {
	for sig := range signals {
		if sig == nil {
			continue
		}

		// Filter: only process PropertiesChanged for systemd unit interface
		// with ActiveState/SubState changes.
		if len(sig.Body) < 2 {
			continue
		}
		iface, ok := sig.Body[0].(string)
		if !ok || iface != "org.freedesktop.systemd1.Unit" {
			continue
		}
		changed, ok := sig.Body[1].(map[string]dbus.Variant)
		if !ok {
			continue
		}
		_, hasActive := changed["ActiveState"]
		_, hasSub := changed["SubState"]
		if !hasActive && !hasSub {
			continue
		}

		w.mu.Lock()
		subs := w.pathSubs[sig.Path]
		for _, s := range subs {
			select {
			case s.ch <- struct{}{}:
			default:
			}
		}
		w.mu.Unlock()
	}
}

// unitObjectPath converts a systemd unit name to a D-Bus object path.
func unitObjectPath(unitName string) dbus.ObjectPath {
	return dbus.ObjectPath("/org/freedesktop/systemd1/unit/" + escapeUnitName(unitName))
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
