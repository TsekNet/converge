package sysctl

import "fmt"

// Sysctl manages a Linux kernel parameter. Check reads from /proc/sys/,
// Apply writes the value and optionally persists it to /etc/sysctl.d/.
type Sysctl struct {
	Key      string
	Value    string
	Persist  bool
	Critical bool
}

func New(key, value string) *Sysctl {
	return &Sysctl{Key: key, Value: value, Persist: true}
}

func (s *Sysctl) ID() string       { return fmt.Sprintf("sysctl:%s", s.Key) }
func (s *Sysctl) String() string   { return fmt.Sprintf("Sysctl %s = %s", s.Key, s.Value) }
func (s *Sysctl) IsCritical() bool { return s.Critical }
