package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID int) (*Role, error) {
	if !guard.CanDeleteRoles() {
		return nil, errors.Tracef(app.ErrUnauthorised)
	}

	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	err = s.repo.RemoveRole(ctx, roleID)

	return role, errors.Tracef(err)
}
