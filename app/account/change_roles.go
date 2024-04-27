package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
)

type ChangeRolesGuard interface {
	CanChangeRoles(userID string) bool
}

func (s *Service) ChangeRoles(ctx context.Context, guard ChangeRolesGuard, userID string, roleIDs, grants, denials []string) error {
	var input struct {
		userID  UserID
		roleIDs []RoleID
		grants  []Permission
		denials []Permission
	}
	{
		if !guard.CanChangeRoles(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
		}

		for _, roleID := range roleIDs {
			id, err := s.repo.ParseRoleID(roleID)
			if err != nil {
				errs.Set("role ids", err)
			}

			input.roleIDs = append(input.roleIDs, id)
		}

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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	var roles []*Role
	if input.roleIDs != nil {
		roles = make([]*Role, len(input.roleIDs))

		for i, roleID := range input.roleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID.String())
			if err != nil {
				return fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	if err := user.ChangeRoles(roles, input.grants, input.denials); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
