package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ChangeRolesGuard interface {
	CanChangeRoles(userID int) bool
}

type ChangeRolesInput struct {
	UserID  int
	RoleIDs []int
	Grants  []Permission
	Denials []Permission
}

func (s *Service) ChangeRolesValidate(guard ChangeRolesGuard, userID int, roleIDs []int, grants, denials []string) (ChangeRolesInput, error) {
	var input ChangeRolesInput

	if !guard.CanChangeRoles(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.UserID = userID
	input.RoleIDs = roleIDs

	if grants != nil {
		input.Grants = make([]Permission, len(grants))

		for i, permission := range grants {
			input.Grants[i], err = NewPermission(permission)
			if err != nil {
				errs.Set("grants", err)
			}
		}
	}
	if denials != nil {
		input.Denials = make([]Permission, len(denials))

		for i, permission := range denials {
			input.Denials[i], err = NewPermission(permission)
			if err != nil {
				errs.Set("denials", err)
			}
		}
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) ChangeRoles(ctx context.Context, guard ChangeRolesGuard, userID int, roleIDs []int, grants, denials []string) (*User, error) {
	input, err := s.ChangeRolesValidate(guard, userID, roleIDs, grants, denials)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	var roles []*Role
	if input.RoleIDs != nil {
		roles = make([]*Role, len(input.RoleIDs))

		for i, roleID := range input.RoleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID)
			if err != nil {
				return nil, fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	user.ChangeRoles(roles, input.Grants, input.Denials)

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
