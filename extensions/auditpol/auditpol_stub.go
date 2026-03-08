//go:build linux || darwin

package auditpol

import (
	"context"
	"fmt"

	"github.com/TsekNet/converge/extensions"
)

type AuditPolicy struct {
	Subcategory string
	Success     bool
	Failure     bool
	Critical    bool
}

func New(subcategory string, success, failure bool) *AuditPolicy {
	return &AuditPolicy{Subcategory: subcategory, Success: success, Failure: failure}
}

func (a *AuditPolicy) ID() string       { return fmt.Sprintf("auditpol:%s", a.Subcategory) }
func (a *AuditPolicy) String() string   { return fmt.Sprintf("AuditPolicy %s", a.Subcategory) }
func (a *AuditPolicy) IsCritical() bool { return a.Critical }

func (a *AuditPolicy) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: true}, nil
}

func (a *AuditPolicy) Apply(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "skipped (not windows)"}, nil
}
