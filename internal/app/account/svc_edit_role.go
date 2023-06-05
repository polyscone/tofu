package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo"
)

type EditRoleGuard interface {
	CanEditRoles() bool
}

func (s *Service) EditRole(ctx context.Context, guard EditRoleGuard, roleID int, name, description string) (*Role, error) {
	var input struct {
		name        text.Name
		description text.OptionalDesc
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
		if input.description, err = text.NewOptionalDesc(description); err != nil {
			errs.Set("description", err)
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
	role.Description = input.description.String()

	err = s.repo.SaveRole(ctx, role)

	var conflicts *repo.ConflictError
	if errors.As(err, &conflicts) {
		return nil, conflicts.Tracef(app.ErrConflictingInput)
	}

	return role, errors.Tracef(err)
}
