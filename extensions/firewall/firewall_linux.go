//go:build linux

package firewall

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/TsekNet/converge/extensions"
	"github.com/TsekNet/converge/internal/version"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

const (
	tableName = version.App
	chainIn   = "input"
	chainOut  = "output"
)

// Check determines whether a matching nftables rule exists.
func (f *Firewall) Check(_ context.Context) (*extensions.State, error) {
	exists, err := f.ruleExists()
	if err != nil {
		return nil, fmt.Errorf("check firewall rule %q: %w", f.Name, err)
	}
	return checkResult(f.Name, exists, f.State != "absent")
}

// Apply creates or removes the nftables rule.
func (f *Firewall) Apply(_ context.Context) (*extensions.Result, error) {
	if f.State == "absent" {
		return f.removeRule()
	}
	return f.addRule()
}

func (f *Firewall) addRule() (*extensions.Result, error) {
	c, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("nftables conn: %w", err)
	}

	// Delete existing rule first to prevent duplicates on repeated Apply.
	// Ignore "table/chain not found" errors (first run).
	if err := f.deleteExistingRules(c); err != nil {
		return nil, fmt.Errorf("cleanup existing rules: %w", err)
	}

	table := ensureTable(c)
	chain := f.ensureChain(c, table)
	f.buildRule(c, table, chain)

	if err := c.Flush(); err != nil {
		return nil, fmt.Errorf("nftables flush: %w", err)
	}

	return resultChanged("added")
}

func (f *Firewall) removeRule() (*extensions.Result, error) {
	c, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("nftables conn: %w", err)
	}

	if err := f.deleteExistingRules(c); err != nil {
		return nil, err
	}

	if err := c.Flush(); err != nil {
		return nil, fmt.Errorf("nftables flush: %w", err)
	}

	return resultChanged("removed")
}

func (f *Firewall) findMatchingRules(c *nftables.Conn) ([]*nftables.Rule, error) {
	table := &nftables.Table{Name: tableName, Family: nftables.TableFamilyIPv4}
	chain := &nftables.Chain{Name: f.chainName(), Table: table}

	rules, err := c.GetRules(table, chain)
	if err != nil {
		// Table/chain doesn't exist yet: no matching rules.
		return nil, nil
	}

	var matched []*nftables.Rule
	for _, rule := range rules {
		if hasUserData(rule, f.Name) {
			matched = append(matched, rule)
		}
	}
	return matched, nil
}

func (f *Firewall) deleteExistingRules(c *nftables.Conn) error {
	matched, err := f.findMatchingRules(c)
	if err != nil {
		return err
	}
	for _, rule := range matched {
		if err := c.DelRule(rule); err != nil {
			return fmt.Errorf("delete rule: %w", err)
		}
	}
	return nil
}

func (f *Firewall) ruleExists() (bool, error) {
	c, err := nftables.New()
	if err != nil {
		return false, fmt.Errorf("nftables conn: %w", err)
	}
	matched, err := f.findMatchingRules(c)
	if err != nil {
		return false, err
	}
	return len(matched) > 0, nil
}

func ensureTable(c *nftables.Conn) *nftables.Table {
	return c.AddTable(&nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyIPv4,
	})
}

func (f *Firewall) ensureChain(c *nftables.Conn, table *nftables.Table) *nftables.Chain {
	hookType := nftables.ChainHookInput
	if f.Direction == "outbound" {
		hookType = nftables.ChainHookOutput
	}
	// Policy is ACCEPT: converge rules are additive filters on top of the
	// host's existing baseline. A missing rule falls through to the default
	// policy, not to DROP. This matches the Windows/macOS behavior.
	chainPolicy := nftables.ChainPolicyAccept
	return c.AddChain(&nftables.Chain{
		Name:     f.chainName(),
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  hookType,
		Priority: nftables.ChainPriorityFilter,
		Policy:   &chainPolicy,
	})
}

func (f *Firewall) buildRule(c *nftables.Conn, table *nftables.Table, chain *nftables.Chain) {
	var exprs []expr.Any

	proto := unix.IPPROTO_TCP
	if strings.EqualFold(f.Protocol, "udp") {
		proto = unix.IPPROTO_UDP
	}

	exprs = append(exprs,
		&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{byte(proto)},
		},
	)

	// Offset 2 = destination port in TCP/UDP header. For inbound traffic this
	// is the local port; for outbound it is the remote port. Port always means
	// "destination port of the packet" across all platforms.
	if f.Port > 0 {
		exprs = append(exprs,
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.BigEndian.PutUint16(uint16(f.Port)),
			},
		)
	}

	if f.Source != "" {
		exprs = append(exprs, matchAddr(f.Source, 12)...) // Source IP offset in IPv4.
	}

	if f.Dest != "" {
		exprs = append(exprs, matchAddr(f.Dest, 16)...) // Dest IP offset in IPv4.
	}

	verdict := expr.VerdictAccept
	if f.Action == "block" {
		verdict = expr.VerdictDrop
	}
	exprs = append(exprs, &expr.Verdict{Kind: verdict})

	c.AddRule(&nftables.Rule{
		Table:    table,
		Chain:    chain,
		Exprs:    exprs,
		UserData: []byte(f.Name),
	})
}

// matchAddr builds nftables expressions to match an IP or CIDR address.
func matchAddr(addr string, offset uint32) []expr.Any {
	// Try CIDR first.
	if _, ipNet, err := net.ParseCIDR(addr); err == nil {
		ones, _ := ipNet.Mask.Size()
		return []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       offset,
				Len:          4,
			},
			&expr.Bitwise{
				SourceRegister: 1,
				DestRegister:   1,
				Len:            4,
				Mask:           net.CIDRMask(ones, 32),
				Xor:            []byte{0, 0, 0, 0},
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ipNet.IP.To4(),
			},
		}
	}

	// Bare IP.
	if ip := net.ParseIP(addr); ip != nil {
		return []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       offset,
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip.To4(),
			},
		}
	}

	return nil
}

func (f *Firewall) chainName() string {
	if f.Direction == "outbound" {
		return chainOut
	}
	return chainIn
}

func hasUserData(rule *nftables.Rule, name string) bool {
	return string(rule.UserData) == name
}
