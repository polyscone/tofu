package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository"
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
			return nil, app.ErrUnauthorised
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

	role := NewRole(input.name, input.description, input.permissions)

	if err := s.repo.AddRole(ctx, role); err != nil {
		var conflict *repository.ConflictError
		if errors.As(err, &conflict) {
			return nil, fmt.Errorf("add role: %w: %w", app.ErrConflictingInput, conflict)
		}

		return nil, fmt.Errorf("add role: %w", err)
	}

	return role, nil
}
