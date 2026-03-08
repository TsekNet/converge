//go:build linux || darwin

package registry

import (
	"context"
	"fmt"

	"github.com/TsekNet/converge/extensions"
)

type Registry struct {
	Key      string
	Value    string
	Type     string
	Data     any
	State    string // "present" (default) or "absent"
	Critical bool
}

func New(key string) *Registry {
	return &Registry{Key: key, State: "present"}
}

func (r *Registry) ID() string       { return fmt.Sprintf("registry:%s\\%s", r.Key, r.Value) }
func (r *Registry) String() string   { return fmt.Sprintf("Registry %s\\%s", r.Key, r.Value) }
func (r *Registry) IsCritical() bool { return r.Critical }

func (r *Registry) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: true}, nil
}

func (r *Registry) Apply(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "skipped (not windows)"}, nil
}
