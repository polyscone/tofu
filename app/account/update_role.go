package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type UpdateRoleGuard interface {
	CanUpdateRoles() bool
}

type UpdateRoleInput struct {
	RoleID      int
	Name        RoleName
	Description RoleDesc
	Permissions []Permission
}

func (s *Service) UpdateRoleValidate(guard UpdateRoleGuard, roleID int, name, description string, permissions []string) (UpdateRoleInput, error) {
	var input UpdateRoleInput

	if !guard.CanUpdateRoles() {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.RoleID = roleID

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

func (s *Service) UpdateRole(ctx context.Context, guard UpdateRoleGuard, roleID int, name, description string, permissions []string) (*Role, error) {
	input, err := s.UpdateRoleValidate(guard, roleID, name, description, permissions)
	if err != nil {
		return nil, err
	}

	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}

	role.Name = input.Name.String()
	role.Description = input.Description.String()

	role.Permissions = nil
	for _, permission := range input.Permissions {
		role.Permissions = append(role.Permissions, permission.String())
	}

	if err := s.repo.SaveRole(ctx, role); err != nil {
		var conflict *app.ConflictError
		if errors.As(err, &conflict) {
			return nil, fmt.Errorf("save role: %w: %w", app.ErrConflict, conflict)
		}

		return nil, fmt.Errorf("save role: %w", err)
	}

	return role, nil
}
