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
	Grants  []string
	Denials []string
}

type ChangeRolesData struct {
	UserID  int
	RoleIDs []int
	Grants  []Permission
	Denials []Permission
}

func (s *Service) ChangeRolesValidate(guard ChangeRolesGuard, input ChangeRolesInput) (ChangeRolesData, error) {
	var data ChangeRolesData

	if !guard.CanChangeRoles(input.UserID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = input.UserID
	data.RoleIDs = input.RoleIDs

	if input.Grants != nil {
		data.Grants = make([]Permission, len(input.Grants))

		for i, permission := range input.Grants {
			data.Grants[i], err = NewPermission(permission)
			if err != nil {
				errs.Set("grants", err)
			}
		}
	}
	if input.Denials != nil {
		data.Denials = make([]Permission, len(input.Denials))

		for i, permission := range input.Denials {
			data.Denials[i], err = NewPermission(permission)
			if err != nil {
				errs.Set("denials", err)
			}
		}
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) ChangeRoles(ctx context.Context, guard ChangeRolesGuard, input ChangeRolesInput) (*User, error) {
	data, err := s.ChangeRolesValidate(guard, input)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	var roles []*Role
	if data.RoleIDs != nil {
		roles = make([]*Role, len(data.RoleIDs))

		for i, roleID := range data.RoleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID)
			if err != nil {
				return nil, fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	user.ChangeRoles(roles, data.Grants, data.Denials)

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
