package service

import "fmt"

// Service manages a system service. Check/Apply are in platform-specific files
// (systemd on Linux, SCM on Windows, launchd stub on macOS).
type Service struct {
	Name        string
	State       string // "running" or "stopped"
	Enable      bool
	StartupType string // "auto", "delayed-auto", "manual", "disabled" (Windows SCM)
	InitSystem  string
	Critical    bool
}

func New(name, state string, enable bool, initSystem string) *Service {
	return &Service{Name: name, State: state, Enable: enable, InitSystem: initSystem}
}

func (s *Service) ID() string       { return fmt.Sprintf("service:%s", s.Name) }
func (s *Service) String() string   { return fmt.Sprintf("Service %s", s.Name) }
func (s *Service) IsCritical() bool { return s.Critical }
