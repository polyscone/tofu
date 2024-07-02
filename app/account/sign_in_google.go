package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type GoogleSignInBehaviour byte

const (
	GoogleSignInOnly GoogleSignInBehaviour = iota
	GoogleAllowSignUp
	GoogleAllowSignUpActivate
)

var ErrGoogleSignUpDisabled = errors.New("Google sign up disabled")

func (s *Service) signInWithGoogle(ctx context.Context, email string, behaviour GoogleSignInBehaviour) (bool, error) {
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
			return false, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	var signedIn bool
	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	switch {
	case err == nil:
		if err := user.SignInWithGoogle(s.system); err != nil {
			return false, err
		}

		signedIn = true

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return false, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		if behaviour == GoogleSignInOnly {
			return false, ErrGoogleSignUpDisabled
		}

		user = NewUser(input.email)

		user.SignUpWithGoogle(s.system)

		if behaviour == GoogleAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return false, err
			}

			if err := user.SignInWithGoogle(s.system); err != nil {
				return false, err
			}

			signedIn = true
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			var conflict *app.ConflictError
			if errors.As(err, &conflict) {
				return false, fmt.Errorf("add user: %w: %w", app.ErrConflict, conflict)
			}

			return false, fmt.Errorf("add user: %w", err)
		}

	default:
		return false, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return signedIn, nil
}

func (s *Service) SignInWithGoogle(ctx context.Context, email string, behaviour GoogleSignInBehaviour) (bool, error) {
	signedIn, err := s.signInWithGoogle(ctx, email, behaviour)
	if err != nil {
		return signedIn, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return signedIn, nil
}
