package pkg

import (
	"context"
	"fmt"

	"github.com/TsekNet/converge/extensions"
)

// PackageManager abstracts OS-specific package operations (apt, brew, choco, etc.).
type PackageManager interface {
	Name() string
	IsInstalled(ctx context.Context, name string) (bool, error)
	Install(ctx context.Context, name string) error
	Remove(ctx context.Context, name string) error
}

type Package struct {
	PkgName     string
	State       string // "present" or "absent"
	Manager     PackageManager
	ManagerName string
	Critical    bool
}

func New(name, state, managerName string) *Package {
	return &Package{
		PkgName:     name,
		State:       state,
		ManagerName: managerName,
		Manager:     detectManager(managerName),
	}
}

func (p *Package) ID() string       { return fmt.Sprintf("package:%s", p.PkgName) }
func (p *Package) String() string   { return fmt.Sprintf("Package %s", p.PkgName) }
func (p *Package) IsCritical() bool { return p.Critical }

func (p *Package) Check(ctx context.Context) (*extensions.State, error) {
	if p.Manager == nil {
		return nil, fmt.Errorf("no package manager detected (expected %s)", p.ManagerName)
	}

	installed, err := p.Manager.IsInstalled(ctx, p.PkgName)
	if err != nil {
		return nil, fmt.Errorf("check %s: %w", p.PkgName, err)
	}

	wantPresent := p.State == "present"

	if installed == wantPresent {
		return &extensions.State{InSync: true}, nil
	}

	action := "add"
	to := "install via " + p.Manager.Name()
	if !wantPresent {
		action = "remove"
		to = "remove via " + p.Manager.Name()
	}

	return &extensions.State{
		InSync: false,
		Changes: []extensions.Change{
			{Property: p.PkgName, To: to, Action: action},
		},
	}, nil
}

func (p *Package) Apply(ctx context.Context) (*extensions.Result, error) {
	if p.Manager == nil {
		return nil, fmt.Errorf("no package manager detected")
	}

	var err error
	var msg string
	if p.State == "present" {
		err = p.Manager.Install(ctx, p.PkgName)
		msg = "installed"
	} else {
		err = p.Manager.Remove(ctx, p.PkgName)
		msg = "removed"
	}
	if err != nil {
		return nil, err
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: msg}, nil
}

// detectManager maps the platform-detected package manager name to its implementation.
func detectManager(name string) PackageManager {
	switch name {
	case "apt":
		return &aptManager{}
	case "brew":
		return &brewManager{}
	case "choco":
		return &chocoManager{}
	case "dnf":
		return &dnfManager{}
	case "yum":
		return &yumManager{}
	case "zypper":
		return &zypperManager{}
	case "apk":
		return &apkManager{}
	case "pacman":
		return &pacmanManager{}
	case "winget":
		return &wingetManager{}
	default:
		return nil
	}
}
