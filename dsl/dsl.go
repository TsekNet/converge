// Package dsl provides the public SDK for building desired-state
// blueprints that compile into a single cross-platform binary.
package dsl

import (
	"os"
	"time"
)

// Blueprint is a function that declares desired system state.
type Blueprint func(r *Run)

type ResourceState string

const (
	Present ResourceState = "present"
	Absent  ResourceState = "absent"
)

type ServiceState string

const (
	Running ServiceState = "running"
	Stopped ServiceState = "stopped"
)

// ResourceMeta holds common metadata shared by all Opts structs.
type ResourceMeta struct {
	DependsOn []string
	Critical  bool
	Noop      bool    // skip Apply, only Check (per-resource dry-run)
	Retry     int     // per-resource max retries (0 = use daemon default)
	Limit     float64 // per-resource rate limit (0 = use daemon default)
	AutoEdge  *bool   // nil = enabled (default), false = disable auto-edges for this resource
	AutoGroup *bool   // nil = enabled (default), false = disable auto-grouping for this resource
}

type FileOpts struct {
	Content string
	Mode    os.FileMode
	Owner   string
	Group   string
	Append  bool
	Meta    ResourceMeta
}

type PackageOpts struct {
	State ResourceState
	Meta  ResourceMeta
}

type ServiceOpts struct {
	State       ServiceState
	Enable      bool
	StartupType string // "auto", "delayed-auto", "manual", "disabled" (Windows SCM)
	Meta        ResourceMeta
}

type ExecOpts struct {
	Command    string
	Args       []string
	OnlyIf     string
	Dir        string
	Env        []string
	Retries    int
	RetryDelay time.Duration
	Meta       ResourceMeta
}

type UserOpts struct {
	Groups []string
	Shell  string
	Home   string
	System bool
	Meta   ResourceMeta
}

type RegistryOpts struct {
	Value string
	Type  string
	Data  any
	State ResourceState // Present (default) or Absent
	Meta  ResourceMeta
}

type SecurityPolicyOpts struct {
	Category string // "password" or "lockout"
	Key      string
	Value    string
	Meta     ResourceMeta
}

type AuditPolicyOpts struct {
	Subcategory string
	Success     bool
	Failure     bool
	Meta        ResourceMeta
}

type SysctlOpts struct {
	Value   string
	Persist bool
	Meta    ResourceMeta
}

type PlistOpts struct {
	Key   string
	Value any
	Type  string // "bool", "int", "float", "string"
	Host  bool   // true = /Library/Preferences (system-wide), false = ~/Library/Preferences
	Meta  ResourceMeta
}

type FirewallOpts struct {
	Port      int
	Protocol  string // "tcp" or "udp"
	Direction string // "inbound" or "outbound"
	Action    string // "allow" or "block"
	Source    string // Optional source address/CIDR
	Dest      string // Optional destination address/CIDR
	State     ResourceState
	Meta      ResourceMeta
}
