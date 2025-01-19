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
	NewPassword      string
	NewPasswordCheck string
}

type ChoosePasswordData struct {
	UserID           int
	NewPassword      Password
	NewPasswordCheck Password
}

func (s *Service) ChoosePasswordValidate(guard ChoosePasswordGuard, input ChoosePasswordInput) (ChoosePasswordData, error) {
	var data ChoosePasswordData

	if !guard.CanChoosePassword(input.UserID) {
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

func (s *Service) ChoosePassword(ctx context.Context, guard ChoosePasswordGuard, input ChoosePasswordInput) (*User, error) {
	data, err := s.ChoosePasswordValidate(guard, input)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChoosePassword(data.NewPassword, s.hasher); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
