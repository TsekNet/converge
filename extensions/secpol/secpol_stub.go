//go:build linux || darwin

package secpol

import (
	"context"
	"fmt"

	"github.com/TsekNet/converge/extensions"
)

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

func (s *SecurityPolicy) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: true}, nil
}

func (s *SecurityPolicy) Apply(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "skipped (not windows)"}, nil
}
