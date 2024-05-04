package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

func (s *Service) signInWithTOTP(ctx context.Context, userID, totp string) error {
	var input struct {
		userID UserID
		totp   TOTP
	}
	{
		var err error
		var errs errsx.Map

		if input.userID, err = s.repo.ParseUserID(userID); err != nil {
			errs.Set("user id", err)
		}
		if input.totp, err = NewTOTP(totp); err != nil {
			errs.Set("totp", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByID(ctx, input.userID.String())
	if err != nil {
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := user.SignInWithTOTP(s.system, input.totp); err != nil {
		return err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}

func (s *Service) SignInWithTOTP(ctx context.Context, userID, totp string) error {
	err := s.signInWithTOTP(ctx, userID, totp)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return nil
}
