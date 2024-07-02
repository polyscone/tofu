package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

func (s *Service) signInWithTOTP(ctx context.Context, userID int, totp string) (*User, error) {
	var input struct {
		userID int
		totp   TOTP
	}
	{
		var err error
		var errs errsx.Map

		input.userID = userID

		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}

		if errs != nil {
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithTOTP(s.system, input.totp); err != nil {
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
