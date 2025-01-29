package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type CreateRoleGuard interface {
	CanCreateRoles() bool
}

type CreateRoleInput struct {
	Name        string
	Description string
	Permissions []string
}

type CreateRoleData struct {
	Name        RoleName
	Description RoleDesc
	Permissions []Permission
}

func (s *Service) CreateRoleValidate(guard CreateRoleGuard, input CreateRoleInput) (CreateRoleData, error) {
	var data CreateRoleData

	if !guard.CanCreateRoles() {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	if data.Name, err = NewRoleName(input.Name); err != nil {
		errs.Set("name", err)
	}
	if data.Description, err = NewRoleDesc(input.Description); err != nil {
		errs.Set("description", err)
	}
	if input.Permissions != nil {
		data.Permissions = make([]Permission, len(input.Permissions))

		for i, permission := range input.Permissions {
			data.Permissions[i], err = NewPermission(permission)
			if err != nil {
				errs.Set("permissions", err)
			}
		}
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) CreateRole(ctx context.Context, guard CreateRoleGuard, input CreateRoleInput) (*Role, error) {
	data, err := s.CreateRoleValidate(guard, input)
	if err != nil {
		return nil, err
	}

	role := NewRole(data.Name, data.Description, data.Permissions)

	if err := s.repo.AddRole(ctx, role); err != nil {
		return nil, fmt.Errorf("add role: %w", err)
	}

	return role, nil
}
