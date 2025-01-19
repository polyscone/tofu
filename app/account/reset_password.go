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
	NewPassword      string
	NewPasswordCheck string
}

type ResetPasswordData struct {
	UserID           int
	NewPassword      Password
	NewPasswordCheck Password
}

func (s *Service) ResetPasswordValidate(guard ResetPasswordGuard, input ResetPasswordInput) (ResetPasswordData, error) {
	var data ResetPasswordData

	if !guard.CanResetPassword(input.UserID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.NewPasswordCheck, _ = NewPassword(input.NewPasswordCheck)

	data.UserID = input.UserID

	if data.NewPassword, err = NewPassword(input.NewPassword); err != nil {
		errs.Set("new password", err)
	} else if !data.NewPassword.Equal(data.NewPasswordCheck) {
		errs.Set("new password", "passwords do not match")
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) ResetPassword(ctx context.Context, guard ResetPasswordGuard, input ResetPasswordInput) (*User, error) {
	data, err := s.ResetPasswordValidate(guard, input)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ResetPassword(data.NewPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
