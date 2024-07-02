package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type ChangePasswordGuard interface {
	CanChangePassword(userID int) bool
}

func (s *Service) ChangePassword(ctx context.Context, guard ChangePasswordGuard, userID int, oldPassword, newPassword, newPasswordCheck string) error {
	var input struct {
		userID           int
		oldPassword      Password
		newPassword      Password
		newPasswordCheck Password
	}
	{
		if !guard.CanChangePassword(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		newPasswordCheck, _ := NewPassword(newPasswordCheck)

		input.userID = userID

		if input.oldPassword, err = NewPassword(oldPassword); err != nil {
			errs.Set("old password", err)
		}
		if input.newPassword, err = NewPassword(newPassword); err != nil {
			errs.Set("new password", err)
		} else if !input.newPassword.Equal(newPasswordCheck) {
			errs.Set("new password", "passwords do not match")
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChangePassword(input.oldPassword, input.newPassword, s.hasher); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
