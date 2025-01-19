package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type FacebookSignInBehavior byte

const (
	FacebookSignInOnly FacebookSignInBehavior = iota
	FacebookAllowSignUp
	FacebookAllowSignUpActivate
)

var ErrFacebookSignUpDisabled = errors.New("Facebook sign up disabled")

type SignInWithFacebookData struct {
	Email Email
}

func (s *Service) SignInWithFacebookValidate(email string) (SignInWithFacebookData, error) {
	var data SignInWithFacebookData
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

func (s *Service) signInWithFacebook(ctx context.Context, email string, behavior FacebookSignInBehavior) (*User, bool, error) {
	data, err := s.SignInWithFacebookValidate(email)
	if err != nil {
		return nil, false, err
	}

	var signedIn bool
	user, err := s.repo.FindUserByEmail(ctx, data.Email.String())
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
		if behavior == FacebookSignInOnly {
			return nil, false, ErrFacebookSignUpDisabled
		}

		user = NewUser(data.Email)

		user.SignUpWithFacebook(s.system)

		if behavior == FacebookAllowSignUpActivate {
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

	s.broker.Flush(ctx, &user.Events)

	return user, signedIn, nil
}

func (s *Service) SignInWithFacebook(ctx context.Context, email string, behavior FacebookSignInBehavior) (*User, bool, error) {
	user, signedIn, err := s.signInWithFacebook(ctx, email, behavior)
	if err != nil {
		return user, signedIn, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, signedIn, nil
}
