package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type ResetPasswordGuard interface {
	CanResetPassword(userID int) bool
}

func (s *Service) ResetPassword(ctx context.Context, guard ResetPasswordGuard, userID int, newPassword, newPasswordCheck string) error {
	var input struct {
		userID           int
		email            text.Email
		newPassword      Password
		newPasswordCheck Password
	}
	{
		if !guard.CanResetPassword(userID) {
			return errors.Tracef(app.ErrUnauthorised)
		}

		var err error
		var errs errors.Map

		newPasswordCheck, _ := NewPassword(newPasswordCheck)

		input.userID = userID

		if input.newPassword, err = NewPassword(newPassword); err != nil {
			errs.Set("new password", err)
		} else if !input.newPassword.Equal(newPasswordCheck) {
			errs.Set("new password", "passwords do not match")
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := user.ResetPassword(input.newPassword, s.hasher); err != nil {
		return errors.Tracef(err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
