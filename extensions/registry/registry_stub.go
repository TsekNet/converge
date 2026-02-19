//go:build linux || darwin

package registry

import (
	"context"
	"fmt"

	"github.com/TsekNet/converge/extensions"
)

// Registry is a no-op stub on non-Windows platforms.
type Registry struct {
	Key      string
	Value    string
	Type     string
	Data     interface{}
	Critical bool
}

func New(key string) *Registry {
	return &Registry{Key: key}
}

func (r *Registry) ID() string       { return fmt.Sprintf("registry:%s", r.Key) }
func (r *Registry) String() string    { return fmt.Sprintf("Registry %s", r.Key) }
func (r *Registry) IsCritical() bool { return r.Critical }

func (r *Registry) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: true}, nil
}

func (r *Registry) Apply(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "skipped (not windows)"}, nil
}
