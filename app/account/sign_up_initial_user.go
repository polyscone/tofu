package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type SignUpInitialUserInput struct {
	Email         Email
	Password      Password
	PasswordCheck Password
	RoleIDs       []int
}

func (s *Service) SignUpInitialUserValidate(email, password, passwordCheck string, roleIDs []int) (SignUpInitialUserInput, error) {
	var input SignUpInitialUserInput
	var err error
	var errs errsx.Map

	input.PasswordCheck, _ = NewPassword(passwordCheck)

	input.RoleIDs = roleIDs

	if input.Email, err = NewEmail(email); err != nil {
		errs.Set("email", err)
	}
	if input.Password, err = NewPassword(password); err != nil {
		errs.Set("password", err)
	} else if !input.Password.Equal(input.PasswordCheck) {
		errs.Set("password", "passwords do not match")
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) signUpInitialUser(ctx context.Context, email, password, passwordCheck string, roleIDs []int) (*User, error) {
	input, err := s.SignUpInitialUserValidate(email, password, passwordCheck, roleIDs)
	if err != nil {
		return nil, err
	}

	var roles []*Role
	if input.RoleIDs != nil {
		roles = make([]*Role, len(input.RoleIDs))

		for i, roleID := range input.RoleIDs {
			role, err := s.repo.FindRoleByID(ctx, roleID)
			if err != nil {
				return nil, fmt.Errorf("find role by id: %w", err)
			}

			roles[i] = role
		}
	}

	userCount, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if userCount != 0 {
		return nil, errors.New("cannot sign up initial user when other users already exist")
	}

	user := NewUser(input.Email)

	if err := user.SignUpAsInitialUser(s.system, roles, input.Password, s.hasher); err != nil {
		return nil, fmt.Errorf("sign up: %w", err)
	}

	if err := s.repo.AddUser(ctx, user); err != nil {
		return nil, fmt.Errorf("add user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}

func (s *Service) SignUpInitialUser(ctx context.Context, email, password, passwordCheck string, roleIDs []int) (*User, error) {
	user, err := s.signUpInitialUser(ctx, email, password, passwordCheck, roleIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
