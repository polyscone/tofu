package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func (s *Service) SignUpInitialUser(ctx context.Context, email, password, passwordCheck string) error {
	var input struct {
		email         Email
		password      Password
		passwordCheck Password
	}
	{
		var err error
		var errs errsx.Map

		input.passwordCheck, _ = NewPassword(passwordCheck)

		if input.email, err = NewEmail(email); err != nil {
			errs.Set("email", err)
		}
		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		} else if !input.password.Equal(input.passwordCheck) {
			errs.Set("password", "passwords do not match")
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	userCount, err := s.repo.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if userCount != 0 {
		return errors.New("cannot sign up initial user when other users already exist")
	}

	user := NewUser(input.email)

	roles := []*Role{SuperRole}
	if err := user.SignUpAsInitialUser(s.system, roles, input.password, s.hasher); err != nil {
		return fmt.Errorf("sign up: %w", err)
	}

	if err := s.repo.AddUser(ctx, user); err != nil {
		return fmt.Errorf("add user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
