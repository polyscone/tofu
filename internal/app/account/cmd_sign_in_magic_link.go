package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository"
)

type MagicLinkSignInBehaviour byte

const (
	MagicLinkSignInOnly MagicLinkSignInBehaviour = iota
	MagicLinkAllowSignUp
	MagicLinkAllowSignUpActivate
)

var ErrMagicLinkSignUpDisabled = errors.New("magic link sign up disabled")

func (s *Service) SignInWithMagicLink(ctx context.Context, email string, behaviour MagicLinkSignInBehaviour) (bool, error) {
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
		if err := user.SignInWithMagicLink(s.system); err != nil {
			return false, err
		}

		signedIn = true

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return false, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, repository.ErrNotFound):
		if behaviour == MagicLinkSignInOnly {
			return false, ErrMagicLinkSignUpDisabled
		}

		user = NewUser(input.email)
		if err := user.SignUpWithMagicLink(s.system); err != nil {
			return false, err
		}

		if behaviour == MagicLinkAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return false, err
			}

			if err := user.SignInWithMagicLink(s.system); err != nil {
				return false, err
			}

			signedIn = true
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			var conflict *repository.ConflictError
			if errors.As(err, &conflict) {
				return false, fmt.Errorf("add user: %w: %w", app.ErrConflictingInput, conflict)
			}

			return false, fmt.Errorf("add user: %w", err)
		}

	default:
		return false, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return signedIn, nil
}