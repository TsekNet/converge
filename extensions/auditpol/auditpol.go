package auditpol

import "fmt"

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
