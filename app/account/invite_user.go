package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type InviteUserGuard interface {
	CanInviteUsers() bool
}

type InviteUserInput struct {
	Email Email
}

func (s *Service) InviteUserValidate(guard InviteUserGuard, email string) (InviteUserInput, error) {
	var input InviteUserInput

	if !guard.CanInviteUsers() {
		return input, app.ErrForbidden
	}

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

func (s *Service) InviteUser(ctx context.Context, guard InviteUserGuard, email string) (*User, error) {
	input, err := s.InviteUserValidate(guard, email)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByEmail(ctx, input.Email.String())
	switch {
	case err == nil:
		if err := user.Invite(s.system); err != nil {
			return nil, fmt.Errorf("invite existing user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		user = NewUser(input.Email)

		if err := user.Invite(s.system); err != nil {
			return nil, fmt.Errorf("invite new user: %w", err)
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			return nil, fmt.Errorf("add user: %w", err)
		}

	default:
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
