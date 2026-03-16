//go:build windows

package firewall

import (
	"context"
	"fmt"
	"strings"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const firewallRulesKey = `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\FirewallRules`

// Check determines whether a matching firewall rule exists with correct content.
func (f *Firewall) Check(_ context.Context) (*extensions.State, error) {
	exists, contentMatch, err := f.ruleState()
	if err != nil {
		return nil, fmt.Errorf("check firewall rule %q: %w", f.Name, err)
	}

	wantPresent := f.State != "absent"

	if wantPresent {
		if exists && contentMatch {
			return &extensions.State{InSync: true}, nil
		}
		action := "add"
		if exists {
			action = "modify"
		}
		return &extensions.State{
			InSync: false,
			Changes: []extensions.Change{{
				Property: "rule",
				From:     boolToState(exists),
				To:       "present",
				Action:   action,
			}},
		}, nil
	}

	return checkResult(f.Name, exists, false)
}

// Apply creates or removes the firewall rule via the registry.
func (f *Firewall) Apply(_ context.Context) (*extensions.Result, error) {
	if f.State == "absent" {
		return f.removeRule()
	}
	return f.addRule()
}

func (f *Firewall) registryName() string {
	return "converge-" + f.Name
}

func (f *Firewall) withRulesKey(access uint32, fn func(k registry.Key) error) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, firewallRulesKey, access)
	if err != nil {
		return fmt.Errorf("open firewall rules key: %w", err)
	}
	defer k.Close()
	return fn(k)
}

func (f *Firewall) addRule() (*extensions.Result, error) {
	ruleData := f.buildRuleString()
	name := f.registryName()

	if err := f.withRulesKey(registry.SET_VALUE, func(k registry.Key) error {
		return k.SetStringValue(name, ruleData)
	}); err != nil {
		return nil, fmt.Errorf("set firewall rule %q: %w", f.Name, err)
	}

	if err := notifyFirewallChange(); err != nil {
		return nil, fmt.Errorf("notify firewall service: %w", err)
	}

	return resultChanged("added")
}

func (f *Firewall) removeRule() (*extensions.Result, error) {
	name := f.registryName()
	var notFound bool

	if err := f.withRulesKey(registry.SET_VALUE, func(k registry.Key) error {
		err := k.DeleteValue(name)
		if err == registry.ErrNotExist {
			notFound = true
			return nil
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("delete firewall rule %q: %w", f.Name, err)
	}

	if notFound {
		return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "already absent"}, nil
	}

	if err := notifyFirewallChange(); err != nil {
		return nil, fmt.Errorf("notify firewall service: %w", err)
	}

	return resultChanged("removed")
}

func (f *Firewall) ruleState() (exists, contentMatch bool, err error) {
	err = f.withRulesKey(registry.QUERY_VALUE, func(k registry.Key) error {
		val, _, e := k.GetStringValue(f.registryName())
		if e == registry.ErrNotExist {
			return nil
		}
		if e != nil {
			return e
		}
		exists = true
		contentMatch = val == f.buildRuleString()
		return nil
	})
	return
}

var winAction = map[string]string{"block": "Block", "allow": "Allow"}
var winDirection = map[string]string{"outbound": "Out", "inbound": "In"}
var winProtocol = map[string]int{"udp": 17, "tcp": 6}

// buildRuleString creates the pipe-delimited rule format used by Windows Firewall.
// Port always means destination port of the packet.
func (f *Firewall) buildRuleString() string {
	parts := []string{
		"v2.33",
		"Action=" + winAction[f.Action],
		"Active=TRUE",
		"Dir=" + winDirection[f.Direction],
		fmt.Sprintf("Protocol=%d", winProtocol[f.Protocol]),
	}

	if f.Port > 0 {
		if f.Direction == "inbound" {
			parts = append(parts, fmt.Sprintf("LPort=%d", f.Port))
		} else {
			parts = append(parts, fmt.Sprintf("RPort=%d", f.Port))
		}
	}

	// For inbound: Source=remote, Dest=local.
	// For outbound: Source=local, Dest=remote.
	if f.Source != "" {
		if f.Direction == "inbound" {
			parts = append(parts, "RA4="+f.Source)
		} else {
			parts = append(parts, "LA4="+f.Source)
		}
	}
	if f.Dest != "" {
		if f.Direction == "inbound" {
			parts = append(parts, "LA4="+f.Dest)
		} else {
			parts = append(parts, "RA4="+f.Dest)
		}
	}

	parts = append(parts, "Name="+f.Name, "Desc=Managed by converge")
	return strings.Join(parts, "|")
}

func notifyFirewallChange() error {
	// The Windows Firewall service (mpssvc) caches rules from the registry.
	// To force an immediate reload, we send PARAMCHANGE then fall back to
	// stop/start if that doesn't work. On most Windows versions, PARAMCHANGE
	// is sufficient. The registry write itself ensures the rule persists
	// across reboots regardless.
	const serviceName = "mpssvc"
	m, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return nil // SCManager not accessible, rule takes effect on next restart
	}
	defer windows.CloseServiceHandle(m)

	access := uint32(windows.SERVICE_PAUSE_CONTINUE | windows.SERVICE_STOP | windows.SERVICE_START | windows.SERVICE_QUERY_STATUS)
	s, err := windows.OpenService(m, windows.StringToUTF16Ptr(serviceName), access)
	if err != nil {
		return nil // service not accessible
	}
	defer windows.CloseServiceHandle(s)

	// Try PARAMCHANGE first (works on most Windows versions).
	var status windows.SERVICE_STATUS
	err = windows.ControlService(s, windows.SERVICE_CONTROL_PARAMCHANGE, &status)
	if err == nil {
		return nil
	}

	// Fallback: stop and restart the service to force a full reload.
	_ = windows.ControlService(s, windows.SERVICE_CONTROL_STOP, &status)
	// Wait briefly for the service to stop.
	for range 20 {
		if err := windows.QueryServiceStatus(s, &status); err != nil {
			break
		}
		if status.CurrentState == windows.SERVICE_STOPPED {
			break
		}
		windows.SleepEx(250, false)
	}
	if err := windows.StartService(s, 0, nil); err != nil {
		// If we can't restart, the rule still persists in the registry
		// and will take effect on next boot.
		return nil
	}
	return nil
}
