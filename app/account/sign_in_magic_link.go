package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type MagicLinkSignInBehaviour byte

const (
	MagicLinkSignInOnly MagicLinkSignInBehaviour = iota
	MagicLinkAllowSignUp
	MagicLinkAllowSignUpActivate
)

var ErrMagicLinkSignUpDisabled = errors.New("magic link sign up disabled")

type SignInWithMagicLinkInput struct {
	Email Email
}

func (s *Service) SignInWithMagicLinkValidate(email string) (SignInWithMagicLinkInput, error) {
	var input SignInWithMagicLinkInput

	var err error
	var errs errsx.Map

	if input.Email, err = NewEmail(email); err != nil {
		errs.Set("email", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) signInWithMagicLink(ctx context.Context, email string, behaviour MagicLinkSignInBehaviour) (*User, bool, error) {
	input, err := s.SignInWithMagicLinkValidate(email)
	if err != nil {
		return nil, false, err
	}

	var signedIn bool
	user, err := s.repo.FindUserByEmail(ctx, input.Email.String())
	switch {
	case err == nil:
		if err := user.SignInWithMagicLink(s.system); err != nil {
			return nil, false, err
		}

		signedIn = true

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return nil, false, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		if behaviour == MagicLinkSignInOnly {
			return nil, false, ErrMagicLinkSignUpDisabled
		}

		user = NewUser(input.Email)

		user.SignUpWithMagicLink(s.system)

		if behaviour == MagicLinkAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return nil, false, err
			}

			if err := user.SignInWithMagicLink(s.system); err != nil {
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

func (s *Service) SignInWithMagicLink(ctx context.Context, email string, behaviour MagicLinkSignInBehaviour) (*User, bool, error) {
	user, signedIn, err := s.signInWithMagicLink(ctx, email, behaviour)
	if err != nil {
		return user, signedIn, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, signedIn, nil
}
