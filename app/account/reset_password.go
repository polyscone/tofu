package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type ResetPasswordGuard interface {
	CanResetPassword(userID int) bool
}

func (s *Service) ResetPassword(ctx context.Context, guard ResetPasswordGuard, userID int, newPassword, newPasswordCheck string) (*User, error) {
	var input struct {
		userID           int
		email            Email
		newPassword      Password
		newPasswordCheck Password
	}
	{
		if !guard.CanResetPassword(userID) {
			return nil, app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		newPasswordCheck, _ := NewPassword(newPasswordCheck)

		input.userID = userID

		if input.newPassword, err = NewPassword(newPassword); err != nil {
			errs.Set("new password", err)
		} else if !input.newPassword.Equal(newPasswordCheck) {
			errs.Set("new password", "passwords do not match")
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ResetPassword(input.newPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
