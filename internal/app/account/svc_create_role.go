package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
)

type CreateRoleGuard interface {
	CanCreateRoles() bool
}

func (s *Service) CreateRole(ctx context.Context, guard CreateRoleGuard, name, description string, permissions []string) (*Role, error) {
	var input struct {
		name        RoleName
		description RoleDesc
		permissions []Permission
	}
	{
		if !guard.CanCreateRoles() {
			return nil, errors.Tracef(app.ErrUnauthorised)
		}

		var err error
		var errs errors.Map

		if input.name, err = NewRoleName(name); err != nil {
			errs.Set("name", err)
		}
		if input.description, err = NewRoleDesc(description); err != nil {
			errs.Set("description", err)
		}
		if permissions != nil {
			input.permissions = make([]Permission, len(permissions))

			for i, permission := range permissions {
				input.permissions[i], err = NewPermission(permission)
				if err != nil {
					errs.Set("permissions", err)
				}
			}
		}

		if errs != nil {
			return nil, errs.Tracef(app.ErrMalformedInput)
		}
	}

	role := NewRole(input.name, input.description, input.permissions)

	if err := s.store.AddRole(ctx, role); err != nil {
		var conflicts *repo.ConflictError
		if errors.As(err, &conflicts) {
			return nil, conflicts.Tracef(app.ErrConflictingInput)
		}

		return nil, errors.Tracef(err)
	}

	return role, nil
}
