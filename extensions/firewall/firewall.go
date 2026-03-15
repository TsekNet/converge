// Package firewall manages host firewall rules across platforms.
// Windows uses the registry-based FirewallRules API, Linux uses nftables
// via netlink (inet family, IPv4 only), and macOS uses pf via anchors.
package firewall

import (
	"fmt"
	"net"
	"regexp"

	"github.com/TsekNet/converge/extensions"
)

// Firewall manages a single named firewall rule.
type Firewall struct {
	Name      string // Human-readable rule name (used as identifier).
	Port      int
	Protocol  string // "tcp" or "udp".
	Direction string // "inbound" or "outbound".
	Action    string // "allow" or "block".
	Source    string // Optional source IP or CIDR. Empty = any.
	Dest      string // Optional destination IP or CIDR. Empty = any.
	State     string // "present" (default) or "absent".
	Critical  bool
}

var validName = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)

// validName must never allow '|' or '=' to prevent Windows firewall rule injection.
var validProtocols = map[string]bool{"tcp": true, "udp": true}
var validDirections = map[string]bool{"inbound": true, "outbound": true}
var validActions = map[string]bool{"allow": true, "block": true}
var validStates = map[string]bool{"present": true, "absent": true}

func New(name string, port int, protocol, direction, action string) *Firewall {
	f := &Firewall{
		Name:      name,
		Port:      port,
		Protocol:  protocol,
		Direction: direction,
		Action:    action,
		State:     "present",
	}
	if err := f.Validate(); err != nil {
		panic(fmt.Sprintf("converge: invalid Firewall: %v", err))
	}
	return f
}

// Validate checks all fields for correctness and rejects values that
// could cause rule injection or produce silently broken rules.
func (f *Firewall) Validate() error {
	if f.Name == "" || !validName.MatchString(f.Name) {
		return fmt.Errorf("name must match %s, got %q", validName.String(), f.Name)
	}
	if f.Port < 1 || f.Port > 65535 {
		return fmt.Errorf("port must be 1-65535, got %d", f.Port)
	}
	if !validProtocols[f.Protocol] {
		return fmt.Errorf("protocol must be tcp or udp, got %q", f.Protocol)
	}
	if !validDirections[f.Direction] {
		return fmt.Errorf("direction must be inbound or outbound, got %q", f.Direction)
	}
	if !validActions[f.Action] {
		return fmt.Errorf("action must be allow or block, got %q", f.Action)
	}
	if !validStates[f.State] {
		return fmt.Errorf("state must be present or absent, got %q", f.State)
	}
	if f.Source != "" {
		if err := validateAddr(f.Source); err != nil {
			return fmt.Errorf("source: %w", err)
		}
	}
	if f.Dest != "" {
		if err := validateAddr(f.Dest); err != nil {
			return fmt.Errorf("dest: %w", err)
		}
	}
	return nil
}

// validateAddr accepts IPv4 addresses and CIDRs only. IPv6 is not yet
// supported by the nftables and pf rule builders.
func validateAddr(addr string) error {
	if ip := net.ParseIP(addr); ip != nil {
		if ip.To4() == nil {
			return fmt.Errorf("IPv6 not supported, got %q", addr)
		}
		return nil
	}
	if _, ipNet, err := net.ParseCIDR(addr); err == nil {
		if ipNet.IP.To4() == nil {
			return fmt.Errorf("IPv6 CIDR not supported, got %q", addr)
		}
		return nil
	}
	return fmt.Errorf("must be a valid IPv4 address or CIDR, got %q", addr)
}

// checkResult builds a State from whether the rule exists and whether we want it.
// Shared across all platform Check implementations.
func checkResult(name string, exists, wantPresent bool) (*extensions.State, error) {
	if exists == wantPresent {
		return &extensions.State{InSync: true}, nil
	}
	action := "add"
	if !wantPresent {
		action = "remove"
	}
	return &extensions.State{
		InSync: false,
		Changes: []extensions.Change{{
			Property: "rule",
			From:     boolToState(!wantPresent),
			To:       boolToState(wantPresent),
			Action:   action,
		}},
	}, nil
}

func (f *Firewall) ID() string       { return fmt.Sprintf("firewall:%s", f.Name) }
func (f *Firewall) String() string   { return fmt.Sprintf("Firewall %s (%s/%d %s)", f.Name, f.Protocol, f.Port, f.Action) }
func (f *Firewall) IsCritical() bool { return f.Critical }

// resultChanged builds a success Result with the given message.
// Shared across all platform Apply implementations.
func resultChanged(message string) (*extensions.Result, error) {
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: message}, nil
}

// boolToState converts a boolean to "present" or "absent".
func boolToState(b bool) string {
	if b {
		return "present"
	}
	return "absent"
}
