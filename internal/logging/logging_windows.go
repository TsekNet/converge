//go:build windows

package logging

import (
	"github.com/google/deck"
	"github.com/google/deck/backends/eventlog"
	"golang.org/x/sys/windows/registry"
)

func init() {
	ensureEventSource()
	evt, err := eventlog.Init(AppID)
	if err != nil {
		return
	}
	deck.Add(evt)
}

// ensureEventSource creates the Windows Event Log source registry key
// if it doesn't exist. This is normally done by the MSI installer,
// but we do it here as a fallback for manual installs.
func ensureEventSource() {
	keyPath := `SYSTEM\CurrentControlSet\Services\EventLog\Application\` + AppID
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, keyPath, registry.SET_VALUE)
	if err != nil {
		return // not admin, can't register
	}
	defer k.Close()
	// EventMessageFile points to the binary itself for message formatting.
	// TypesSupported enables Information, Warning, Error events.
	k.SetExpandStringValue("EventMessageFile", `%SystemRoot%\System32\EventCreate.exe`)
	k.SetDWordValue("TypesSupported", 7) // EVENTLOG_ERROR|WARNING|INFORMATION
}
