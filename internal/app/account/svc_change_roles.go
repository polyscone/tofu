package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
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
			return errors.Tracef(app.ErrUnauthorised)
		}

		for _, roleID := range roleIDs {
			if roleID == SuperRole.ID && !guard.CanAssignSuperRole(userID) {
				return errors.Tracef(app.ErrUnauthorised)
			}
		}

		var err error
		var errs errors.Map

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
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.store.FindUserByID(ctx, input.userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return errors.Tracef(app.ErrMalformedInput, err)
		}

		return errors.Tracef(err)
	}

	var roles []*Role
	if roleIDs != nil {
		roles = make([]*Role, len(roleIDs))

		for i, roleID := range roleIDs {
			role, err := s.store.FindRoleByID(ctx, roleID)
			if err != nil {
				if errors.Is(err, repo.ErrNotFound) {
					return errors.Tracef(app.ErrMalformedInput, err)
				}

				return errors.Tracef(err)
			}

			roles[i] = role
		}
	}

	if err := user.ChangeRoles(roles, input.grants, input.denials); err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
