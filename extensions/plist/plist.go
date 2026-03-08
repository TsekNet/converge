package plist

import "fmt"

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
