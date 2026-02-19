//go:build darwin

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
	cmd := exec.CommandContext(ctx, "dscl", ".", "-create", fmt.Sprintf("/Users/%s", u.Name))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dscl create %s: %s: %w", u.Name, strings.TrimSpace(string(out)), err)
	}

	if u.Shell != "" {
		exec.CommandContext(ctx, "dscl", ".", "-create", fmt.Sprintf("/Users/%s", u.Name), "UserShell", u.Shell).Run()
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Created"}, nil
}

func (u *User) modifyUser(ctx context.Context) (*extensions.Result, error) {
	if u.Shell != "" {
		cmd := exec.CommandContext(ctx, "dscl", ".", "-create", fmt.Sprintf("/Users/%s", u.Name), "UserShell", u.Shell)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("dscl modify shell %s: %s: %w", u.Name, strings.TrimSpace(string(out)), err)
		}
		return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Modified"}, nil
	}
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "OK"}, nil
}

func shellForUser(u *osuser.User) string {
	cmd := exec.Command("dscl", ".", "-read", fmt.Sprintf("/Users/%s", u.Name), "UserShell")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), " ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
