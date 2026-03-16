//go:build windows

package firewall

import (
	"context"
	"fmt"
	"runtime"

	"github.com/TsekNet/converge/extensions"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// NET_FW constants matching the Windows Firewall COM enums.
const (
	netFwIPProtocolTCP = 6
	netFwIPProtocolUDP = 17
	netFwRuleDirectionIn  = 1
	netFwRuleDirectionOut = 2
	netFwActionBlock = 0
	netFwActionAllow = 1
)

var directionMap = map[string]int32{"inbound": netFwRuleDirectionIn, "outbound": netFwRuleDirectionOut}
var actionMap = map[string]int32{"block": netFwActionBlock, "allow": netFwActionAllow}
var protocolMap = map[string]int32{"tcp": netFwIPProtocolTCP, "udp": netFwIPProtocolUDP}

// Check determines whether a matching firewall rule exists with correct properties.
func (f *Firewall) Check(_ context.Context) (*extensions.State, error) {
	exists, match, err := f.withCOM(func(rules *ole.IDispatch) (bool, bool, error) {
		rule, err := oleutil.CallMethod(rules, "Item", f.Name)
		if err != nil {
			return false, false, nil // rule not found
		}
		r := rule.ToIDispatch()
		defer r.Release()

		return true, f.ruleMatches(r), nil
	})
	if err != nil {
		return nil, fmt.Errorf("check firewall rule %q: %w", f.Name, err)
	}

	wantPresent := f.State != "absent"

	if wantPresent {
		if exists && match {
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

// Apply creates or removes the firewall rule via the Windows Firewall COM API.
func (f *Firewall) Apply(_ context.Context) (*extensions.Result, error) {
	if f.State == "absent" {
		return f.removeRule()
	}
	return f.addRule()
}

func (f *Firewall) addRule() (*extensions.Result, error) {
	_, _, err := f.withCOM(func(rules *ole.IDispatch) (bool, bool, error) {
		// Remove existing rule if any (atomic replace).
		oleutil.CallMethod(rules, "Remove", f.Name)

		// Create a new rule via COM.
		unknown, err := oleutil.CreateObject("HNetCfg.FWRule")
		if err != nil {
			return false, false, fmt.Errorf("create FWRule: %w", err)
		}
		defer unknown.Release()

		rule, err := unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return false, false, fmt.Errorf("query IDispatch: %w", err)
		}
		defer rule.Release()

		oleutil.PutProperty(rule, "Name", f.Name)
		oleutil.PutProperty(rule, "Description", "Managed by converge")
		oleutil.PutProperty(rule, "Protocol", protocolMap[f.Protocol])
		oleutil.PutProperty(rule, "Direction", directionMap[f.Direction])
		oleutil.PutProperty(rule, "Action", actionMap[f.Action])
		oleutil.PutProperty(rule, "Enabled", true)

		if f.Port > 0 {
			oleutil.PutProperty(rule, "LocalPorts", fmt.Sprintf("%d", f.Port))
		}
		if f.Source != "" {
			oleutil.PutProperty(rule, "RemoteAddresses", f.Source)
		}
		if f.Dest != "" {
			oleutil.PutProperty(rule, "LocalAddresses", f.Dest)
		}

		if _, err := oleutil.CallMethod(rules, "Add", rule); err != nil {
			return false, false, fmt.Errorf("add rule: %w", err)
		}

		return false, false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("add firewall rule %q: %w", f.Name, err)
	}
	return resultChanged("added")
}

func (f *Firewall) removeRule() (*extensions.Result, error) {
	_, _, err := f.withCOM(func(rules *ole.IDispatch) (bool, bool, error) {
		if _, err := oleutil.CallMethod(rules, "Remove", f.Name); err != nil {
			// Rule not found is not an error.
			return false, false, nil
		}
		return false, false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("remove firewall rule %q: %w", f.Name, err)
	}
	return resultChanged("removed")
}

// ruleMatches checks if an existing rule has the expected properties.
func (f *Firewall) ruleMatches(r *ole.IDispatch) bool {
	proto, _ := oleutil.GetProperty(r, "Protocol")
	dir, _ := oleutil.GetProperty(r, "Direction")
	act, _ := oleutil.GetProperty(r, "Action")
	enabled, _ := oleutil.GetProperty(r, "Enabled")
	ports, _ := oleutil.GetProperty(r, "LocalPorts")

	if proto.Val != int64(protocolMap[f.Protocol]) {
		return false
	}
	if dir.Val != int64(directionMap[f.Direction]) {
		return false
	}
	if act.Val != int64(actionMap[f.Action]) {
		return false
	}
	if enabled.Val == 0 {
		return false
	}
	if f.Port > 0 {
		portStr := fmt.Sprintf("%d", f.Port)
		if ports.ToString() != portStr {
			return false
		}
	}
	return true
}

// withCOM initializes COM, gets the firewall rules collection, and calls fn.
// Handles all COM lifecycle (init, create, release).
func (f *Firewall) withCOM(fn func(rules *ole.IDispatch) (bool, bool, error)) (bool, bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return false, false, fmt.Errorf("CoInitializeEx: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
	if err != nil {
		return false, false, fmt.Errorf("create FwPolicy2: %w", err)
	}
	defer unknown.Release()

	policy, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return false, false, fmt.Errorf("query IDispatch: %w", err)
	}
	defer policy.Release()

	rulesResult, err := oleutil.GetProperty(policy, "Rules")
	if err != nil {
		return false, false, fmt.Errorf("get Rules: %w", err)
	}
	rules := rulesResult.ToIDispatch()
	defer rules.Release()

	return fn(rules)
}
