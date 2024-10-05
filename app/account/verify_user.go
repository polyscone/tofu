package account

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
)

type VerifyUserBehaviour byte

const (
	VerifyUserOnly VerifyUserBehaviour = iota
	VerifyUserActivate
)

type VerifyUserInput struct {
	Email         Email
	Password      Password
	PasswordCheck Password
}

func (s *Service) VerifyUserValidate(email, password, passwordCheck string) (VerifyUserInput, error) {
	var input VerifyUserInput
	var err error
	var errs errsx.Map

	input.PasswordCheck, _ = NewPassword(passwordCheck)

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

func (s *Service) VerifyUser(ctx context.Context, email, password, passwordCheck string, behaviour VerifyUserBehaviour) (*User, error) {
	input, err := s.VerifyUserValidate(email, password, passwordCheck)
	if err != nil {
		return nil, err
	}

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	user, err := s.repo.FindUserByEmail(ctx, input.Email.String())
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := user.Verify(input.Password, s.hasher); err != nil {
		return nil, err
	}

	if isInvited := !user.InvitedAt.IsZero(); isInvited || behaviour == VerifyUserActivate {
		if err := user.Activate(); err != nil {
			return nil, err
		}
	}

	log.Attempts = 0
	log.LastAttemptAt = time.Time{}

	if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
		return nil, fmt.Errorf("save sign in attempt log: %w", err)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
