package account

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

type VerifyUserBehavior byte

const (
	VerifyUserOnly VerifyUserBehavior = iota
	VerifyUserActivate
)

type VerifyUserInput struct {
	Email         string
	Password      string
	PasswordCheck string
	Behavior      VerifyUserBehavior
}

type VerifyUserData struct {
	Email         Email
	Password      Password
	PasswordCheck Password
	Behavior      VerifyUserBehavior
}

func (s *Service) VerifyUserValidate(input VerifyUserInput) (VerifyUserData, error) {
	var data VerifyUserData
	var err error
	var errs errsx.Map

	data.PasswordCheck, _ = NewPassword(input.PasswordCheck)

	if data.Email, err = NewEmail(input.Email); err != nil {
		errs.Set("email", err)
	}
	if data.Password, err = NewPassword(input.Password); err != nil {
		errs.Set("password", err)
	} else if !data.Password.Equal(data.PasswordCheck) {
		errs.Set("password", "passwords do not match")
	}

	data.Behavior = input.Behavior

	if errs != nil {
		return data, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return data, nil
}

func (s *Service) VerifyUser(ctx context.Context, input VerifyUserInput) (*User, error) {
	data, err := s.VerifyUserValidate(input)
	if err != nil {
		return nil, err
	}

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, data.Email.String())
	if err != nil {
		return nil, fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	user, err := s.repo.FindUserByEmail(ctx, data.Email.String())
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := user.Verify(data.Password, s.hasher); err != nil {
		return nil, err
	}

	if isInvited := !user.InvitedAt.IsZero(); isInvited || data.Behavior == VerifyUserActivate {
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

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
