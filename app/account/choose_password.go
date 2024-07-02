package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type ChoosePasswordGuard interface {
	CanChoosePassword(userID int) bool
}

func (s *Service) ChoosePassword(ctx context.Context, guard ChoosePasswordGuard, userID int, newPassword, newPasswordCheck string) (*User, error) {
	var input struct {
		userID           int
		newPassword      Password
		newPasswordCheck Password
	}
	{
		if !guard.CanChoosePassword(userID) {
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

	if err := user.ChoosePassword(input.newPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
