package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type UpdateRoleGuard interface {
	CanUpdateRoles() bool
}

type UpdateRoleInput struct {
	RoleID      int
	Name        string
	Description string
	Permissions []string
}

type UpdateRoleData struct {
	RoleID      int
	Name        RoleName
	Description RoleDesc
	Permissions []Permission
}

func (s *Service) UpdateRoleValidate(guard UpdateRoleGuard, input UpdateRoleInput) (UpdateRoleData, error) {
	var data UpdateRoleData

	if !guard.CanUpdateRoles() {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.RoleID = input.RoleID

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

func (s *Service) UpdateRole(ctx context.Context, guard UpdateRoleGuard, input UpdateRoleInput) (*Role, error) {
	data, err := s.UpdateRoleValidate(guard, input)
	if err != nil {
		return nil, err
	}

	role, err := s.repo.FindRoleByID(ctx, data.RoleID)
	if err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}

	role.ChangeName(data.Name)
	role.ChangeDescription(data.Description)
	role.ChangePermissions(data.Permissions)

	if err := s.repo.SaveRole(ctx, role); err != nil {
		var conflict *app.ConflictError
		if errors.As(err, &conflict) {
			return nil, fmt.Errorf("save role: %w: %w", app.ErrConflict, conflict)
		}

		return nil, fmt.Errorf("save role: %w", err)
	}

	return role, nil
}
