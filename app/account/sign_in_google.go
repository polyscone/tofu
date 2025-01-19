package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type GoogleSignInBehavior byte

const (
	GoogleSignInOnly GoogleSignInBehavior = iota
	GoogleAllowSignUp
	GoogleAllowSignUpActivate
)

var ErrGoogleSignUpDisabled = errors.New("Google sign up disabled")

type SignInWithGoogleData struct {
	Email Email
}

func (s *Service) SignInWithGoogleValidate(email string) (SignInWithGoogleData, error) {
	var data SignInWithGoogleData
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

func (s *Service) signInWithGoogle(ctx context.Context, email string, behavior GoogleSignInBehavior) (*User, bool, error) {
	data, err := s.SignInWithGoogleValidate(email)
	if err != nil {
		return nil, false, err
	}

	var signedIn bool
	user, err := s.repo.FindUserByEmail(ctx, data.Email.String())
	switch {
	case err == nil:
		if err := user.SignInWithGoogle(s.system); err != nil {
			return nil, false, err
		}

		signedIn = true

		if err := s.repo.SaveUser(ctx, user); err != nil {
			return nil, false, fmt.Errorf("save user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		if behavior == GoogleSignInOnly {
			return nil, false, ErrGoogleSignUpDisabled
		}

		user = NewUser(data.Email)

		user.SignUpWithGoogle(s.system)

		if behavior == GoogleAllowSignUpActivate {
			if err := user.Activate(); err != nil {
				return nil, false, err
			}

			if err := user.SignInWithGoogle(s.system); err != nil {
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

func (s *Service) SignInWithGoogle(ctx context.Context, email string, behavior GoogleSignInBehavior) (*User, bool, error) {
	user, signedIn, err := s.signInWithGoogle(ctx, email, behavior)
	if err != nil {
		return user, signedIn, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, signedIn, nil
}
