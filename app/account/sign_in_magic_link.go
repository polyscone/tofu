package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type MagicLinkSignInBehavior byte

const (
	MagicLinkSignInOnly MagicLinkSignInBehavior = iota
	MagicLinkAllowSignUp
	MagicLinkAllowSignUpActivate
)

var ErrMagicLinkSignUpDisabled = errors.New("magic link sign up disabled")

type SignInWithMagicLinkData struct {
	Email Email
}

func (s *Service) SignInWithMagicLinkValidate(email string) (SignInWithMagicLinkData, error) {
	var data SignInWithMagicLinkData

	var err error
	var errs errsx.Map

	if data.Email, err = NewEmail(email); err != nil {
		errs.Set("email", err)
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) signInWithMagicLink(ctx context.Context, email string, behavior MagicLinkSignInBehavior) (*User, bool, error) {
	data, err := s.SignInWithMagicLinkValidate(email)
	if err != nil {
		return nil, false, err
	}

	var signedIn bool
	user, err := s.repo.FindUserByEmail(ctx, data.Email.String())
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
		if behavior == MagicLinkSignInOnly {
			return nil, false, ErrMagicLinkSignUpDisabled
		}

		user = NewUser(data.Email)

		user.SignUpWithMagicLink(s.system)

		if behavior == MagicLinkAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return nil, false, err
			}

			if err := user.SignInWithMagicLink(s.system); err != nil {
				return nil, false, err
			}

			signedIn = true
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			return nil, false, fmt.Errorf("add user: %w", err)
		}

	default:
		return nil, false, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, signedIn, nil
}

func (s *Service) SignInWithMagicLink(ctx context.Context, email string, behavior MagicLinkSignInBehavior) (*User, bool, error) {
	user, signedIn, err := s.signInWithMagicLink(ctx, email, behavior)
	if err != nil {
		return user, signedIn, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, signedIn, nil
}
