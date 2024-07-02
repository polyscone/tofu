package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID int) (*Role, error) {
	var input struct {
		roleID int
	}
	{
		if !guard.CanDeleteRoles() {
			return nil, app.ErrForbidden
		}

		input.roleID = roleID
	}

	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}

	if err := s.repo.RemoveRole(ctx, input.roleID); err != nil {
		return nil, fmt.Errorf("remove role: %w", err)
	}

	return role, nil
}
