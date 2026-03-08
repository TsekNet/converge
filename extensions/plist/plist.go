package plist

import "fmt"

// Plist manages a macOS preference domain key. Check/Apply use howett.net/plist
// for native binary plist encoding (no defaults command).
type Plist struct {
	Domain   string
	Key      string
	Value    any
	Type     string // "bool", "int", "float", "string"
	Host     bool   // true = /Library/Preferences (system-wide)
	Critical bool
}

func New(domain, key string) *Plist {
	return &Plist{Domain: domain, Key: key}
}

func (p *Plist) ID() string       { return fmt.Sprintf("plist:%s:%s", p.Domain, p.Key) }
func (p *Plist) String() string   { return fmt.Sprintf("Plist %s %s", p.Domain, p.Key) }
func (p *Plist) IsCritical() bool { return p.Critical }
