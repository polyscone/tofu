package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type EditRoleGuard interface {
	CanEditRoles() bool
}

func (s *Service) EditRole(ctx context.Context, guard EditRoleGuard, roleID int, name string) (*Role, error) {
	var input struct {
		name text.Name
	}
	{
		if !guard.CanEditRoles() {
			return nil, errors.Tracef(app.ErrUnauthorised)
		}

		var err error
		var errs errors.Map

		if input.name, err = text.NewName(name); err != nil {
			errs.Set("name", err)
		}

		if errs != nil {
			return nil, errs.Tracef(app.ErrMalformedInput)
		}
	}

	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	role.Name = input.name.String()

	err = s.repo.SaveRole(ctx, role)

	return role, errors.Tracef(err)
}
