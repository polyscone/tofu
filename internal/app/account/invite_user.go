package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type InviteUserGuard interface {
	CanInviteUsers() bool
}

func (s *Service) InviteUser(ctx context.Context, guard InviteUserGuard, email string) error {
	var input struct {
		email Email
	}
	{
		if !guard.CanInviteUsers() {
			return app.ErrForbidden
		}

		var err error
		var errs errsx.Map

		if input.email, err = NewEmail(email); err != nil {
			errs.Set("email", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	switch {
	case err == nil:
		if err := user.Invite(s.system); err != nil {
			return fmt.Errorf("invite existing user: %w", err)
		}

	case errors.Is(err, app.ErrNotFound):
		id, err := s.repo.NextUserID(ctx)
		if err != nil {
			return fmt.Errorf("next user id: %w", err)
		}

		user = NewUser(id, input.email)

		if err := user.Invite(s.system); err != nil {
			return fmt.Errorf("invite new user: %w", err)
		}

		if err := s.repo.AddUser(ctx, user); err != nil {
			return fmt.Errorf("add user: %w", err)
		}

	default:
		return fmt.Errorf("find user by email: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
