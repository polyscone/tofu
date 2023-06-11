package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repo"
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

	user := NewUser(input.email)

	if err := user.SignUp(); err != nil {
		return nil, fmt.Errorf("sign up: %w", err)
	}

	if err := s.store.AddUser(ctx, user); err != nil {
		var conflicts *repo.ConflictError
		if errors.As(err, &conflicts) {
			return nil, fmt.Errorf("add user: %w: %w", app.ErrConflictingInput, conflicts)
		}

		return nil, fmt.Errorf("add user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
