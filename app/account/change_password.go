package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ChangePasswordGuard interface {
	CanChangePassword(userID int) bool
}

type ChangePasswordInput struct {
	UserID           int
	OldPassword      string
	NewPassword      string
	NewPasswordCheck string
}

type ChangePasswordData struct {
	UserID           int
	OldPassword      Password
	NewPassword      Password
	NewPasswordCheck Password
}

func (s *Service) ChangePasswordValidate(guard ChangePasswordGuard, input ChangePasswordInput) (ChangePasswordData, error) {
	var data ChangePasswordData

	if !guard.CanChangePassword(input.UserID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.NewPasswordCheck, _ = NewPassword(input.NewPasswordCheck)

	data.UserID = input.UserID

	if data.OldPassword, err = NewPassword(input.OldPassword); err != nil {
		errs.Set("old password", err)
	}
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

func (s *Service) ChangePassword(ctx context.Context, guard ChangePasswordGuard, input ChangePasswordInput) (*User, error) {
	data, err := s.ChangePasswordValidate(guard, input)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChangePassword(data.OldPassword, data.NewPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
