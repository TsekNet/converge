//go:build linux

package user

import (
	"context"
	"fmt"
	"os/exec"
	osuser "os/user"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

// Apply creates or modifies the user via useradd/usermod.
func (u *User) Apply(ctx context.Context) (*extensions.Result, error) {
	_, err := lookupUser(u.Name)
	if err != nil {
		return u.createUser(ctx)
	}
	return u.modifyUser(ctx)
}

func (u *User) createUser(ctx context.Context) (*extensions.Result, error) {
	var args []string
	if u.System {
		args = append(args, "--system")
	}
	if u.Shell != "" {
		args = append(args, "--shell", u.Shell)
	}
	if u.Home != "" {
		args = append(args, "--home-dir", u.Home, "--create-home")
	}
	if len(u.Groups) > 0 {
		args = append(args, "--groups", strings.Join(u.Groups, ","))
	}
	args = append(args, u.Name)

	cmd := exec.CommandContext(ctx, "useradd", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("useradd %s: %s: %w", u.Name, strings.TrimSpace(string(out)), err)
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Created"}, nil
}

// modifyUser uses --append --groups to add groups without removing existing ones.
func (u *User) modifyUser(ctx context.Context) (*extensions.Result, error) {
	var args []string
	if u.Shell != "" {
		args = append(args, "--shell", u.Shell)
	}
	if len(u.Groups) > 0 {
		args = append(args, "--append", "--groups", strings.Join(u.Groups, ","))
	}
	if len(args) == 0 {
		return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "OK"}, nil
	}
	args = append(args, u.Name)

	cmd := exec.CommandContext(ctx, "usermod", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("usermod %s: %s: %w", u.Name, strings.TrimSpace(string(out)), err)
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Modified"}, nil
}

// shellForUser reads the login shell from /etc/passwd via getent.
func shellForUser(u *osuser.User) string {
	cmd := exec.Command("getent", "passwd", u.Uid)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	fields := strings.Split(strings.TrimSpace(string(out)), ":")
	if len(fields) >= 7 {
		return fields[6]
	}
	return ""
}
