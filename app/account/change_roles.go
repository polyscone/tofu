package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type ChangeRolesGuard interface {
	CanChangeRoles(userID int) bool
}

func (s *Service) ChangeRoles(ctx context.Context, guard ChangeRolesGuard, userID int, roleIDs []int, grants, denials []string) (*User, error) {
	var input struct {
		userID  int
		roleIDs []int
		grants  []Permission
		denials []Permission
	}
	{
		if !guard.CanChangeRoles(userID) {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		input.userID = userID
		input.roleIDs = roleIDs

		if grants != nil {
			input.grants = make([]Permission, len(grants))

			for i, permission := range grants {
				input.grants[i], err = NewPermission(permission)
				if err != nil {
					errs.Set("grants", err)
				}
			}
		}
		if denials != nil {
			input.denials = make([]Permission, len(denials))

			for i, permission := range denials {
				input.denials[i], err = NewPermission(permission)
				if err != nil {
					errs.Set("denials", err)
				}
			}
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	var roles []*Role
	if input.roleIDs != nil {
		roles = make([]*Role, len(input.roleIDs))

		for i, roleID := range input.roleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID)
			if err != nil {
				return nil, fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	user.ChangeRoles(roles, input.grants, input.denials)

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
