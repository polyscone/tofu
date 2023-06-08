package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
)

type EditRoleGuard interface {
	CanEditRoles() bool
}

func (s *Service) EditRole(ctx context.Context, guard EditRoleGuard, roleID int, name, description string, permissions []string) (*Role, error) {
	var input struct {
		name        RoleName
		description RoleDesc
		permissions []Permission
	}
	{
		if !guard.CanEditRoles() {
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

	if _, err := s.store.FindRoleByID(ctx, roleID); err != nil {
		return nil, errors.Tracef(err)
	}

	role := NewRole(input.name, input.description, input.permissions)

	role.ID = roleID

	err := s.store.SaveRole(ctx, role)

	var conflicts *repo.ConflictError
	if errors.As(err, &conflicts) {
		return nil, conflicts.Tracef(app.ErrConflictingInput)
	}

	return role, errors.Tracef(err)
}
