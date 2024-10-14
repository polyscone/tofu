package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type CreateRoleGuard interface {
	CanCreateRoles() bool
}

type CreateRoleInput struct {
	Name        RoleName
	Description RoleDesc
	Permissions []Permission
}

func (s *Service) CreateRoleValidate(guard CreateRoleGuard, name, description string, permissions []string) (CreateRoleInput, error) {
	var input CreateRoleInput

	if !guard.CanCreateRoles() {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	if input.Name, err = NewRoleName(name); err != nil {
		errs.Set("name", err)
	}
	if input.Description, err = NewRoleDesc(description); err != nil {
		errs.Set("description", err)
	}
	if permissions != nil {
		input.Permissions = make([]Permission, len(permissions))

		for i, permission := range permissions {
			input.Permissions[i], err = NewPermission(permission)
			if err != nil {
				errs.Set("permissions", err)
			}
		}
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) CreateRole(ctx context.Context, guard CreateRoleGuard, name, description string, permissions []string) (*Role, error) {
	input, err := s.CreateRoleValidate(guard, name, description, permissions)
	if err != nil {
		return nil, err
	}

	role := NewRole(input.Name, input.Description, input.Permissions)

	if err := s.repo.AddRole(ctx, role); err != nil {
		var conflict *app.ConflictError
		if errors.As(err, &conflict) {
			return nil, fmt.Errorf("add role: %w: %w", app.ErrConflict, conflict)
		}

		return nil, fmt.Errorf("add role: %w", err)
	}

	return role, nil
}
