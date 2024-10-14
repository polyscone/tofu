package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ChoosePasswordGuard interface {
	CanChoosePassword(userID int) bool
}

type ChoosePasswordInput struct {
	UserID           int
	NewPassword      Password
	NewPasswordCheck Password
}

func (s *Service) ChoosePasswordValidate(guard ChoosePasswordGuard, userID int, newPassword, newPasswordCheck string) (ChoosePasswordInput, error) {
	var input ChoosePasswordInput

	if !guard.CanChoosePassword(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.NewPasswordCheck, _ = NewPassword(newPasswordCheck)

	input.UserID = userID

	if input.NewPassword, err = NewPassword(newPassword); err != nil {
		errs.Set("new password", err)
	} else if !input.NewPassword.Equal(input.NewPasswordCheck) {
		errs.Set("new password", "passwords do not match")
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) ChoosePassword(ctx context.Context, guard ChoosePasswordGuard, userID int, newPassword, newPasswordCheck string) (*User, error) {
	input, err := s.ChoosePasswordValidate(guard, userID, newPassword, newPasswordCheck)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChoosePassword(input.NewPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
