package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type SignInWithTOTPInput struct {
	UserID int
	TOTP   TOTP
}

func (s *Service) SignInWithTOTPValidate(userID int, totp string) (SignInWithTOTPInput, error) {
	var input SignInWithTOTPInput
	var err error
	var errs errsx.Map

	input.UserID = userID

	if input.TOTP, err = NewTOTP(totp); err != nil {
		errs.Set("totp", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) signInWithTOTP(ctx context.Context, userID int, totp string) (*User, error) {
	input, err := s.SignInWithTOTPValidate(userID, totp)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithTOTP(s.system, input.TOTP); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}

func (s *Service) SignInWithTOTP(ctx context.Context, userID int, totp string) (*User, error) {
	user, err := s.signInWithTOTP(ctx, userID, totp)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
