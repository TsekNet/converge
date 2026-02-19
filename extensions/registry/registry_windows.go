//go:build windows

package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

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

func (r *Registry) Check(ctx context.Context) (*extensions.State, error) {
	cmd := exec.CommandContext(ctx, "reg", "query", r.Key, "/v", r.Value)
	out, err := cmd.Output()
	if err != nil {
		return &extensions.State{
			InSync: false,
			Changes: []extensions.Change{
				{Property: r.Value, To: fmt.Sprintf("%v", r.Data), Action: "add"},
			},
		}, nil
	}

	if r.Data != nil && !strings.Contains(string(out), fmt.Sprintf("%v", r.Data)) {
		return &extensions.State{
			InSync: false,
			Changes: []extensions.Change{
				{Property: r.Value, To: fmt.Sprintf("%v", r.Data), Action: "modify"},
			},
		}, nil
	}

	return &extensions.State{InSync: true}, nil
}

func (r *Registry) Apply(ctx context.Context) (*extensions.Result, error) {
	regType := "REG_SZ"
	switch strings.ToLower(r.Type) {
	case "dword":
		regType = "REG_DWORD"
	case "qword":
		regType = "REG_QWORD"
	case "expandstring":
		regType = "REG_EXPAND_SZ"
	case "multistring":
		regType = "REG_MULTI_SZ"
	case "binary":
		regType = "REG_BINARY"
	}

	cmd := exec.CommandContext(ctx, "reg", "add", r.Key, "/v", r.Value, "/t", regType, "/d", fmt.Sprintf("%v", r.Data), "/f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("reg add %s: %s: %w", r.Key, strings.TrimSpace(string(out)), err)
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Set"}, nil
}
