//go:build darwin

package service

import (
	"context"

	"github.com/TsekNet/converge/extensions"
)

func (s *Service) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: true}, nil
}

func (s *Service) Apply(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "skipped (launchd not implemented)"}, nil
}
