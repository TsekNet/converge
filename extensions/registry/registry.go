package registry

import "fmt"

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
