package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type ChangeTOTPTelGuard interface {
	CanChangeTOTPTel(userID int) bool
}

type ChangeTOTPTelInput struct {
	UserID int
	NewTel Tel
}

func (s *Service) ChangeTOTPTelValidate(guard ChangeTOTPTelGuard, userID int, newTel string) (ChangeTOTPTelInput, error) {
	var input ChangeTOTPTelInput

	if !guard.CanChangeTOTPTel(userID) {
		return input, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	input.UserID = userID

	if input.NewTel, err = NewTel(newTel); err != nil {
		errs.Set("new phone", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) ChangeTOTPTel(ctx context.Context, guard ChangeTOTPTelGuard, userID int, newTel string) (*User, error) {
	input, err := s.ChangeTOTPTelValidate(guard, userID, newTel)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ChangeTOTPTel(input.NewTel); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
