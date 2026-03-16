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

type FileOpts struct {
	Content   string
	Mode      os.FileMode
	Owner     string
	Group     string
	Append    bool
	Critical  bool
	DependsOn []string
}

type PackageOpts struct {
	State     ResourceState
	Critical  bool
	DependsOn []string
}

type ServiceOpts struct {
	State       ServiceState
	Enable      bool
	StartupType string // "auto", "delayed-auto", "manual", "disabled" (Windows SCM)
	Critical    bool
	DependsOn   []string
}

type ExecOpts struct {
	Command    string
	Args       []string
	OnlyIf     string
	Dir        string
	Env        []string
	Retries    int
	RetryDelay time.Duration
	Critical   bool
	DependsOn  []string
}

type UserOpts struct {
	Groups    []string
	Shell     string
	Home      string
	System    bool
	Critical  bool
	DependsOn []string
}

type RegistryOpts struct {
	Value     string
	Type      string
	Data      any
	State     ResourceState // Present (default) or Absent
	Critical  bool
	DependsOn []string
}

type SecurityPolicyOpts struct {
	Category  string // "password" or "lockout"
	Key       string
	Value     string
	Critical  bool
	DependsOn []string
}

type AuditPolicyOpts struct {
	Subcategory string
	Success     bool
	Failure     bool
	Critical    bool
	DependsOn   []string
}

type SysctlOpts struct {
	Value     string
	Persist   bool
	Critical  bool
	DependsOn []string
}

type PlistOpts struct {
	Key       string
	Value     any
	Type      string // "bool", "int", "float", "string"
	Host      bool   // true = /Library/Preferences (system-wide), false = ~/Library/Preferences
	Critical  bool
	DependsOn []string
}

type FirewallOpts struct {
	Port      int
	Protocol  string // "tcp" or "udp"
	Direction string // "inbound" or "outbound"
	Action    string // "allow" or "block"
	Source    string // Optional source address/CIDR
	Dest      string // Optional destination address/CIDR
	State     ResourceState
	Critical  bool
	DependsOn []string
}
