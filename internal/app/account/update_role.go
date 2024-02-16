package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type UpdateRoleGuard interface {
	CanUpdateRoles() bool
}

func (s *Service) UpdateRole(ctx context.Context, guard UpdateRoleGuard, roleID int, name, description string, permissions []string) (*Role, error) {
	var input struct {
		name        RoleName
		description RoleDesc
		permissions []Permission
	}
	{
		if !guard.CanUpdateRoles() {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

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
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	if _, err := s.repo.FindRoleByID(ctx, roleID); err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}

	role := NewRole(input.name, input.description, input.permissions)

	role.ID = roleID

	err := s.repo.SaveRole(ctx, role)
	if err != nil {
		var conflict *app.ConflictError
		if errors.As(err, &conflict) {
			return nil, fmt.Errorf("save role: %w: %w", app.ErrConflict, conflict)
		}

		return nil, fmt.Errorf("save role: %w", err)
	}

	return role, nil
}
