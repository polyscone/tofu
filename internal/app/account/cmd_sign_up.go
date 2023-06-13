package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository"
)

func (s *Service) SignUp(ctx context.Context, email string) (*User, error) {
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
			return nil, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	switch {
	case err == nil:
		if user.ActivatedAt.IsZero() {
			if err := user.SignUp(); err != nil {
				return nil, fmt.Errorf("sign up existing: %w", err)
			}
		} else {
			conflict := &repository.ConflictError{
				Map: errsx.Map{"email": errors.New("already in use")},
			}

			return nil, fmt.Errorf("sign up existing: %w: %w", app.ErrConflictingInput, conflict)
		}

	case errors.Is(err, repository.ErrNotFound):
		user = NewUser(input.email)

		if err := user.SignUp(); err != nil {
			return nil, fmt.Errorf("sign up: %w", err)
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			var conflict *repository.ConflictError
			if errors.As(err, &conflict) {
				return nil, fmt.Errorf("add user: %w: %w", app.ErrConflictingInput, conflict)
			}

			return nil, fmt.Errorf("add user: %w", err)
		}

	default:
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
