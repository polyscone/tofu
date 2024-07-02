package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID int) error {
	var input struct {
		roleID int
	}
	{
		if !guard.CanDeleteRoles() {
			return app.ErrForbidden
		}

		input.roleID = roleID
	}

	err := s.repo.RemoveRole(ctx, input.roleID)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}

	return nil
}
