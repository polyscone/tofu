package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID int) error {
	if !guard.CanDeleteRoles() {
		return errors.Tracef(app.ErrUnauthorised)
	}

	err := s.repo.RemoveRole(ctx, roleID)

	return errors.Tracef(err)
}
