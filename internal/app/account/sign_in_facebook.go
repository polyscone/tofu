package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type FacebookSignInBehaviour byte

const (
	FacebookSignInOnly FacebookSignInBehaviour = iota
	FacebookAllowSignUp
	FacebookAllowSignUpActivate
)

var ErrFacebookSignUpDisabled = errors.New("Facebook sign up disabled")

func (s *Service) SignInWithFacebook(ctx context.Context, email string, behaviour FacebookSignInBehaviour) (bool, error) {
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
		if err := user.SignInWithFacebook(s.system); err != nil {
			return false, err
		}

		signedIn = true

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return false, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		if behaviour == FacebookSignInOnly {
			return false, ErrFacebookSignUpDisabled
		}

		id, err := s.repo.NextID(ctx)
		if err != nil {
			return false, fmt.Errorf("next id: %w", err)
		}

		user = NewUser(id, input.email)
		if err := user.SignUpWithFacebook(s.system); err != nil {
			return false, err
		}

		if behaviour == FacebookAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return false, err
			}

			if err := user.SignInWithFacebook(s.system); err != nil {
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
