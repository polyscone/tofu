package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func (s *Service) SignUp(ctx context.Context, email string) error {
	var input struct {
		email Email
	}
	{
		var err error
		var errs errsx.Map

		if input.email, err = NewEmail(email); err != nil {
			errs.Set("email", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	switch {
	case err == nil:
		if err := user.SignUp(s.system); err != nil {
			return fmt.Errorf("sign up existing: %w", err)
		}

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		id, err := s.repo.NextUserID(ctx)
		if err != nil {
			return fmt.Errorf("next user id: %w", err)
		}

		user = NewUser(id, input.email)

		if err := user.SignUp(s.system); err != nil {
			return fmt.Errorf("sign up: %w", err)
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			return fmt.Errorf("add user: %w", err)
		}

	default:
		return fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
