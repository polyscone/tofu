package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type CreateRoleGuard interface {
	CanCreateRoles() bool
}

func (s *Service) CreateRole(ctx context.Context, guard CreateRoleGuard, roleID, name, description string, permissions []string) error {
	var input struct {
		roleID      RoleID
		name        RoleName
		description RoleDesc
		permissions []Permission
	}
	{
		if !guard.CanCreateRoles() {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.roleID, err = s.repo.ParseRoleID(roleID); err != nil {
			errs.Set("role id", err)
		}
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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	role := NewRole(input.roleID, input.name, input.description, input.permissions)

	if err := s.repo.AddRole(ctx, role); err != nil {
		var conflict *app.ConflictError
		if errors.As(err, &conflict) {
			return fmt.Errorf("add role: %w: %w", app.ErrConflict, conflict)
		}

		return fmt.Errorf("add role: %w", err)
	}

	return nil
}
