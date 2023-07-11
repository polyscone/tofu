package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository"
)

type GoogleSignInBehaviour byte

const (
	GoogleSignInOnly GoogleSignInBehaviour = iota
	GoogleAllowSignUp
)

var ErrGoogleSignUpDisabled = errors.New("Google sign up disabled")

func (s *Service) SignInWithGoogle(ctx context.Context, email string, behaviour GoogleSignInBehaviour) error {
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
		if err := user.SignInWithGoogle(); err != nil {
			return err
		}

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, repository.ErrNotFound):
		if behaviour == GoogleSignInOnly {
			return ErrGoogleSignUpDisabled
		}

		user = NewUser(input.email)
		if err := user.SignUpWithGoogle(); err != nil {
			return err
		}

		if err := user.SignInWithGoogle(); err != nil {
			return err
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			var conflict *repository.ConflictError
			if errors.As(err, &conflict) {
				return fmt.Errorf("add user: %w: %w", app.ErrConflictingInput, conflict)
			}

			return fmt.Errorf("add user: %w", err)
		}

	default:
		return fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
