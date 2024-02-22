package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func (s *Service) SignUpInitialUser(ctx context.Context, email, password, passwordCheck string, roleIDs []string) error {
	var input struct {
		email         Email
		password      Password
		passwordCheck Password
		roleIDs       []string
	}
	{
		var err error
		var errs errsx.Map

		input.passwordCheck, _ = NewPassword(passwordCheck)
		input.roleIDs = roleIDs

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

	var roles []*Role
	if input.roleIDs != nil {
		roles = make([]*Role, len(input.roleIDs))

		for i, roleID := range input.roleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID)
			if err != nil {
				return fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	userCount, err := s.repo.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if userCount != 0 {
		return errors.New("cannot sign up initial user when other users already exist")
	}

	id, err := s.repo.NextUserID(ctx)
	if err != nil {
		return fmt.Errorf("next id: %w", err)
	}

	user := NewUser(id, input.email)

	if err := user.SignUpAsInitialUser(s.system, roles, input.password, s.hasher); err != nil {
		return fmt.Errorf("sign up: %w", err)
	}

	if err := s.repo.AddUser(ctx, user); err != nil {
		return fmt.Errorf("add user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
