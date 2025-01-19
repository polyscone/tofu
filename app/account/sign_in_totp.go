package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type SignInWithTOTPData struct {
	UserID int
	TOTP   TOTP
}

func (s *Service) SignInWithTOTPValidate(userID int, totp string) (SignInWithTOTPData, error) {
	var data SignInWithTOTPData
	var err error
	var errs errsx.Map

	data.UserID = userID

	if data.TOTP, err = NewTOTP(totp); err != nil {
		errs.Set("totp", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) signInWithTOTP(ctx context.Context, userID int, totp string) (*User, error) {
	data, err := s.SignInWithTOTPValidate(userID, totp)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithTOTP(s.system, data.TOTP); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}

func (s *Service) SignInWithTOTP(ctx context.Context, userID int, totp string) (*User, error) {
	user, err := s.signInWithTOTP(ctx, userID, totp)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
