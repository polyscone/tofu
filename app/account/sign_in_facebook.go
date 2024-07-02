package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type FacebookSignInBehaviour byte

const (
	FacebookSignInOnly FacebookSignInBehaviour = iota
	FacebookAllowSignUp
	FacebookAllowSignUpActivate
)

var ErrFacebookSignUpDisabled = errors.New("Facebook sign up disabled")

func (s *Service) signInWithFacebook(ctx context.Context, email string, behaviour FacebookSignInBehaviour) (*User, bool, error) {
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
			return nil, false, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	var signedIn bool
	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	switch {
	case err == nil:
		if err := user.SignInWithFacebook(s.system); err != nil {
			return nil, false, err
		}

		signedIn = true

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return nil, false, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		if behaviour == FacebookSignInOnly {
			return nil, false, ErrFacebookSignUpDisabled
		}

		user = NewUser(input.email)

		user.SignUpWithFacebook(s.system)

		if behaviour == FacebookAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return nil, false, err
			}

			if err := user.SignInWithFacebook(s.system); err != nil {
				return nil, false, err
			}

			signedIn = true
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			var conflict *app.ConflictError
			if errors.As(err, &conflict) {
				return nil, false, fmt.Errorf("add user: %w: %w", app.ErrConflict, conflict)
			}

			return nil, false, fmt.Errorf("add user: %w", err)
		}

	default:
		return nil, false, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, signedIn, nil
}

func (s *Service) SignInWithFacebook(ctx context.Context, email string, behaviour FacebookSignInBehaviour) (*User, bool, error) {
	user, signedIn, err := s.signInWithFacebook(ctx, email, behaviour)
	if err != nil {
		return user, signedIn, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, signedIn, nil
}
