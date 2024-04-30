package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type ResetPasswordGuard interface {
	CanResetPassword(userID string) bool
}

func (s *Service) ResetPassword(ctx context.Context, guard ResetPasswordGuard, userID string, newPassword, newPasswordCheck string) error {
	var input struct {
		userID           UserID
		email            Email
		newPassword      Password
		newPasswordCheck Password
	}
	{
		if !guard.CanResetPassword(userID) {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		newPasswordCheck, _ := NewPassword(newPasswordCheck)

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
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

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ResetPassword(input.newPassword, s.hasher); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
