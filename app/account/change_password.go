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
	OldPassword      Password
	NewPassword      Password
	NewPasswordCheck Password
}

func (s *Service) ChangePasswordValidate(guard ChangePasswordGuard, userID int, oldPassword, newPassword, newPasswordCheck string) (ChangePasswordInput, error) {
	var input ChangePasswordInput

	if !guard.CanChangePassword(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.NewPasswordCheck, _ = NewPassword(newPasswordCheck)

	input.UserID = userID

	if input.OldPassword, err = NewPassword(oldPassword); err != nil {
		errs.Set("old password", err)
	}
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

func (s *Service) ChangePassword(ctx context.Context, guard ChangePasswordGuard, userID int, oldPassword, newPassword, newPasswordCheck string) (*User, error) {
	input, err := s.ChangePasswordValidate(guard, userID, oldPassword, newPassword, newPasswordCheck)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChangePassword(input.OldPassword, input.NewPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
