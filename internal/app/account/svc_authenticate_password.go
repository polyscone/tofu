package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func (s *Service) AuthenticateWithPassword(ctx context.Context, email, password string) error {
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

	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	if err != nil {
		return errors.Tracef(err)
	}

	_, err = user.AuthenticateWithPassword(input.password, s.hasher)
	if err != nil {
		return errors.Tracef(err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
