package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID int) (*Role, error) {
	if !guard.CanDeleteRoles() {
		return nil, app.ErrUnauthorised
	}

	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}

	err = s.repo.RemoveRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("remove role: %w", err)
	}

	return role, nil
}
