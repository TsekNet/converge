package secpol

import "fmt"

// SecurityPolicy enforces Windows local security policy (password and lockout settings)
// via the NetUserModalsGet/Set Win32 API (no secedit.exe).
type SecurityPolicy struct {
	Category string
	Key      string
	Value    string
	Critical bool
}

func New(category, key, value string) *SecurityPolicy {
	return &SecurityPolicy{Category: category, Key: key, Value: value}
}

func (s *SecurityPolicy) ID() string       { return fmt.Sprintf("secpol:%s:%s", s.Category, s.Key) }
func (s *SecurityPolicy) String() string   { return fmt.Sprintf("SecurityPolicy %s/%s", s.Category, s.Key) }
func (s *SecurityPolicy) IsCritical() bool { return s.Critical }
