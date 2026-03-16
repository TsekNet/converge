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
		"Profile=Public|Profile=Private|Profile=Domain",
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
	const serviceName = "mpssvc"
	m, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return nil
	}
	defer windows.CloseServiceHandle(m)

	s, err := windows.OpenService(m, windows.StringToUTF16Ptr(serviceName), windows.SERVICE_PAUSE_CONTINUE)
	if err != nil {
		return nil
	}
	defer windows.CloseServiceHandle(s)

	var status windows.SERVICE_STATUS
	// PARAMCHANGE signals mpssvc to re-read rules from the registry.
	_ = windows.ControlService(s, windows.SERVICE_CONTROL_PARAMCHANGE, &status)
	return nil
}
