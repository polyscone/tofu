package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type SignUpInitialUserData struct {
	Email         Email
	Password      Password
	PasswordCheck Password
	RoleIDs       []int
}

func (s *Service) SignUpInitialUserValidate(email, password, passwordCheck string, roleIDs []int) (SignUpInitialUserData, error) {
	var data SignUpInitialUserData
	var err error
	var errs errsx.Map

	data.PasswordCheck, _ = NewPassword(passwordCheck)

	data.RoleIDs = roleIDs

	if data.Email, err = NewEmail(email); err != nil {
		errs.Set("email", err)
	}
	if data.Password, err = NewPassword(password); err != nil {
		errs.Set("password", err)
	} else if !data.Password.Equal(data.PasswordCheck) {
		errs.Set("password", "passwords do not match")
	}

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) signUpInitialUser(ctx context.Context, email, password, passwordCheck string, roleIDs []int) (*User, error) {
	data, err := s.SignUpInitialUserValidate(email, password, passwordCheck, roleIDs)
	if err != nil {
		return nil, err
	}

	var roles []*Role
	if data.RoleIDs != nil {
		roles = make([]*Role, len(data.RoleIDs))

		for i, roleID := range data.RoleIDs {
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

	user := NewUser(data.Email)

	if err := user.SignUpAsInitialUser(s.system, roles, data.Password, s.hasher); err != nil {
		return nil, fmt.Errorf("sign up: %w", err)
	}

	if err := s.repo.AddUser(ctx, user); err != nil {
		return nil, fmt.Errorf("add user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}

func (s *Service) SignUpInitialUser(ctx context.Context, email, password, passwordCheck string, roleIDs []int) (*User, error) {
	user, err := s.signUpInitialUser(ctx, email, password, passwordCheck, roleIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
