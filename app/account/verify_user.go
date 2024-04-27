package account

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
)

type VerifyUserBehaviour byte

const (
	VerifyUserOnly VerifyUserBehaviour = iota
	VerifyUserActivate
)

func (s *Service) VerifyUser(ctx context.Context, email, password, passwordCheck string, behaviour VerifyUserBehaviour) error {
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

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	if err := user.Verify(input.password, s.hasher); err != nil {
		return err
	}

	if isInvited := !user.InvitedAt.IsZero(); isInvited || behaviour == VerifyUserActivate {
		if err := user.Activate(); err != nil {
			return err
		}
	}

	log.Attempts = 0
	log.LastAttemptAt = time.Time{}

	if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
		return fmt.Errorf("save sign in attempt log: %w", err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return nil
}
