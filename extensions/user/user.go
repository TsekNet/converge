package user

import (
	"context"
	"fmt"
	"os/user"
	"strings"

	"github.com/TsekNet/converge/extensions"
)

type User struct {
	Name     string
	Groups   []string
	Shell    string
	Home     string
	System   bool
	Critical bool
}

func New(name string, groups []string, shell string) *User {
	return &User{Name: name, Groups: groups, Shell: shell}
}

func (u *User) ID() string       { return fmt.Sprintf("user:%s", u.Name) }
func (u *User) String() string   { return fmt.Sprintf("User %s", u.Name) }
func (u *User) IsCritical() bool { return u.Critical }

func (u *User) Check(_ context.Context) (*extensions.State, error) {
	existing, err := lookupUser(u.Name)
	if err != nil {
		var changes []extensions.Change
		changes = append(changes, extensions.Change{
			Property: "user", To: u.Name, Action: "add",
		})
		if len(u.Groups) > 0 {
			changes = append(changes, extensions.Change{
				Property: "groups", To: strings.Join(u.Groups, ","), Action: "add",
			})
		}
		if u.Shell != "" {
			changes = append(changes, extensions.Change{
				Property: "shell", To: u.Shell, Action: "add",
			})
		}
		return &extensions.State{InSync: false, Changes: changes}, nil
	}

	var changes []extensions.Change

	if u.Shell != "" {
		currentShell := shellForUser(existing)
		if currentShell != "" && currentShell != u.Shell {
			changes = append(changes, extensions.Change{
				Property: "shell", From: currentShell, To: u.Shell, Action: "modify",
			})
		}
	}

	return &extensions.State{InSync: len(changes) == 0, Changes: changes}, nil
}

func lookupUser(name string) (*user.User, error) {
	return user.Lookup(name)
}
