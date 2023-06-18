package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type ChoosePasswordGuard interface {
	CanChoosePassword(userID int) bool
}

func (s *Service) ChoosePassword(ctx context.Context, guard ChoosePasswordGuard, userID int, newPassword, newPasswordCheck string) error {
	var input struct {
		userID           int
		oldPassword      Password
		newPassword      Password
		newPasswordCheck Password
	}
	{
		if !guard.CanChoosePassword(userID) {
			return app.ErrUnauthorised
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
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChoosePassword(input.newPassword, s.hasher); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
