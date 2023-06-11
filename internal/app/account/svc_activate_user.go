package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func (s *Service) ActivateUser(ctx context.Context, email, password, passwordCheck string) error {
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

	user, err := s.store.FindUserByEmail(ctx, input.email.String())
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	superUserCount, err := s.store.CountUsersByRoleID(ctx, SuperRole.ID)
	if err != nil {
		return fmt.Errorf("count users by role id: %w", err)
	}
	if superUserCount == 0 {
		if err := user.ChangeRoles([]*Role{SuperRole}, nil, nil); err != nil {
			return fmt.Errorf("add super role to initial user: %w", err)
		}
	}

	if err := user.Activate(input.password, s.hasher); err != nil {
		return fmt.Errorf("activate user: %w", err)
	}

	if err := s.store.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
