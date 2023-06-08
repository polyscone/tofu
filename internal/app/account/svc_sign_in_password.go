package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func (s *Service) SignInWithPassword(ctx context.Context, email, password string) error {
	var input struct {
		email    text.Email
		password Password
	}
	{
		var err error
		var errs errors.Map

		if input.email, err = text.NewEmail(email); err != nil {
			errs.Set("email", err)
		}
		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return errs.Tracef(app.ErrMalformedInput)
		}
	}

	user, err := s.store.FindUserByEmail(ctx, input.email.String())
	if err != nil {
		// We always hash a password even when we error finding a user to help
		// prevent timing attacks that would allow enumeration of valid emails
		if _, err := s.hasher.EncodedHash(input.password); err != nil {
			return errors.Tracef(err)
		}

		return errors.Tracef(err)
	}

	_, err = user.SignInWithPassword(input.password, s.hasher)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
