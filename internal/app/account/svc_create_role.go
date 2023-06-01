package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo"
)

type CreateRoleGuard interface {
	CanCreateRoles() bool
}

func (s *Service) CreateRole(ctx context.Context, guard CreateRoleGuard, name string) (*Role, error) {
	var input struct {
		name text.Name
	}
	{
		if !guard.CanCreateRoles() {
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

	role := NewRole(input.name)

	if err := s.repo.AddRole(ctx, role); err != nil {
		if errors.Is(err, repo.ErrConflict) {
			errs := errors.Map{"name": errors.New("already in use")}

			return nil, errs.Tracef(app.ErrConflictingInput)
		}

		return nil, errors.Tracef(err)
	}

	return role, nil
}
