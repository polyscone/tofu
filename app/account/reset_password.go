package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ResetPasswordGuard interface {
	CanResetPassword(userID int) bool
}

type ResetPasswordInput struct {
	UserID           int
	Email            Email
	NewPassword      Password
	NewPasswordCheck Password
}

func (s *Service) ResetPasswordValidate(guard ResetPasswordGuard, userID int, newPassword, newPasswordCheck string) (ResetPasswordInput, error) {
	var input ResetPasswordInput

	if !guard.CanResetPassword(userID) {
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

func (s *Service) ResetPassword(ctx context.Context, guard ResetPasswordGuard, userID int, newPassword, newPasswordCheck string) (*User, error) {
	input, err := s.ResetPasswordValidate(guard, userID, newPassword, newPasswordCheck)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ResetPassword(input.NewPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
