package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/uuid"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID string) error {
	var input struct {
		roleID uuid.UUID
	}
	{
		if !guard.CanDeleteRoles() {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.roleID, err = uuid.Parse(roleID); err != nil {
			errs.Set("role id", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	err := s.repo.RemoveRole(ctx, input.roleID.String())
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}

	return nil
}
