//go:build windows

package user

import (
	"context"
	"fmt"
	"os/exec"
	osuser "os/user"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

func (u *User) Apply(ctx context.Context) (*extensions.Result, error) {
	_, err := lookupUser(u.Name)
	if err != nil {
		return u.createUser(ctx)
	}
	return u.modifyUser(ctx)
}

func (u *User) createUser(ctx context.Context) (*extensions.Result, error) {
	cmd := exec.CommandContext(ctx, "net", "user", u.Name, "/add")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("net user %s /add: %s: %w", u.Name, strings.TrimSpace(string(out)), err)
	}

	for _, group := range u.Groups {
		cmd := exec.CommandContext(ctx, "net", "localgroup", group, u.Name, "/add")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("net localgroup %s %s /add: %s: %w", group, u.Name, strings.TrimSpace(string(out)), err)
		}
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Created"}, nil
}

func (u *User) modifyUser(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "OK"}, nil
}

func shellForUser(_ *osuser.User) string {
	return ""
}
