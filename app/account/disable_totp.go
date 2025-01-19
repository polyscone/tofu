package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type DisableTOTPGuard interface {
	CanDisableTOTP(userID int) bool
}

type DisableTOTPData struct {
	UserID   int
	Password Password
}

func (s *Service) DisableTOTPValidate(guard DisableTOTPGuard, userID int, password string) (DisableTOTPData, error) {
	var data DisableTOTPData

	if !guard.CanDisableTOTP(userID) {
		return data, app.ErrForbidden
	}

	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.Password, err = NewPassword(password); err != nil {
		errs.Set("password", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) DisableTOTP(ctx context.Context, guard DisableTOTPGuard, userID int, password string) (*User, error) {
	data, err := s.DisableTOTPValidate(guard, userID, password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.DisableTOTP(data.Password, s.hasher); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			return nil, fmt.Errorf("%w: %w", app.ErrUnauthorized, err)
		}

		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
