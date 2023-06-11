package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func (s *Service) SignInWithPassword(ctx context.Context, email, password string) error {
	var input struct {
		email    Email
		password Password
	}
	{
		var err error
		var errs errsx.Map

		if input.email, err = NewEmail(email); err != nil {
			errs.Set("email", err)
		}
		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.store.FindUserByEmail(ctx, input.email.String())
	if err != nil {
		// We always hash a password even when we error finding a user to help
		// prevent timing attacks that would allow enumeration of valid emails
		if _, err := s.hasher.EncodedHash(input.password); err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		return fmt.Errorf("find user by email: %w", err)
	}

	_, err = user.SignInWithPassword(input.password, s.hasher)
	if err != nil {
		return fmt.Errorf("sign in with password: %w", err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
