package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type ChangeRolesGuard interface {
	CanChangeRoles(userID int) bool
	CanAssignSuperRole(userID int) bool
}

func (s *Service) ChangeRoles(ctx context.Context, guard ChangeRolesGuard, userID int, roleIDs []int, grants, denials []string) error {
	var input struct {
		userID  int
		roleIDs []int
		grants  []Permission
		denials []Permission
	}
	{
		if !guard.CanChangeRoles(userID) {
			return app.ErrUnauthorised
		}

		for _, roleID := range roleIDs {
			if roleID == SuperRole.ID && !guard.CanAssignSuperRole(userID) {
				return app.ErrUnauthorised
			}
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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	var roles []*Role
	if roleIDs != nil {
		roles = make([]*Role, len(roleIDs))

		for i, roleID := range roleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID)
			if err != nil {
				return fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	if err := user.ChangeRoles(roles, input.grants, input.denials); err != nil {
		return fmt.Errorf("change roles: %w", err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
