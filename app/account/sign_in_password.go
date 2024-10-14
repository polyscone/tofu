package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/errsx"
)

const (
	MaxFreeSignInAttempts  = 3
	MaxSignInThrottleDelay = app.SignInThrottleTTL / 2
)

var ErrSignInThrottled = errors.New("sign in throttled")

type SignInThrottleError struct {
	InLast   time.Duration
	Delay    time.Duration
	UnlockAt time.Time
	UnlockIn time.Duration
}

func (t SignInThrottleError) Error() string {
	return fmt.Sprintf("delayed for %v: unlocking at %v", t.Delay, t.UnlockAt.Format("15:04:05 MST"))
}

func (s *Service) CheckSignInThrottle(attempts int, lastAttemptAt time.Time) error {
	if attempts >= MaxFreeSignInAttempts {
		shift := attempts - (MaxFreeSignInAttempts - 1)
		delay := (1 << shift) * time.Second
		if delay > MaxSignInThrottleDelay {
			delay = MaxSignInThrottleDelay
		}

		unlockAt := lastAttemptAt.Add(delay)
		if time.Now().Before(unlockAt) {
			throttle := &SignInThrottleError{
				InLast:   app.SignInThrottleTTL,
				Delay:    delay,
				UnlockAt: unlockAt,
				UnlockIn: time.Until(unlockAt),
			}

			return fmt.Errorf("%w: %w", ErrSignInThrottled, throttle)
		}
	}

	return nil
}

type SignInWithPasswordInput struct {
	Email    Email
	Password Password
}

func (s *Service) SignInWithPasswordValidate(email, password string) (SignInWithPasswordInput, error) {
	var input SignInWithPasswordInput
	var err error
	var errs errsx.Map

	if input.Email, err = NewEmail(email); err != nil {
		errs.Set("email", err)
	}
	if input.Password, err = NewPassword(password); err != nil {
		errs.Set("password", err)
	}

	if errs != nil {
		return input, fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
	}

	return input, nil
}

func (s *Service) signInWithPassword(ctx context.Context, email, password string) (*User, error) {
	input, err := s.SignInWithPasswordValidate(email, password)
	if err != nil {
		return nil, err
	}

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	if err := s.CheckSignInThrottle(log.Attempts, log.LastAttemptAt); err != nil {
		return nil, fmt.Errorf("check sign in throttle: %w", err)
	}

	// Even in the case where a user doesn't exist we always want to log
	// failed sign in attempts to prevent leaking information about which
	// users exist in the system
	//
	// These values are set to their zero values on a successful
	// sign in or account verification
	log.Attempts++
	log.LastAttemptAt = time.Now()

	user, err := s.repo.FindUserByEmail(ctx, input.Email.String())
	if err != nil {
		if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
			return nil, fmt.Errorf("save sign in attempt log: %w", err)
		}

		// Always check a password even when we error finding a user to help
		// avoid leaking info that would allow enumeration of valid emails
		if err := s.hasher.CheckDummyPasswordHash(); err != nil {
			return nil, fmt.Errorf("check dummy password hash: %w", err)
		}

		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if _, err := user.SignInWithPassword(s.system, input.Password, s.hasher); err != nil {
		switch {
		case errors.Is(err, ErrNotVerified),
			errors.Is(err, ErrNotActivated),
			errors.Is(err, ErrInvalidPassword):

			if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
				return nil, fmt.Errorf("save sign in attempt log: %w", err)
			}

			if err := s.repo.SaveUser(ctx, user); err != nil {
				return nil, fmt.Errorf("save user: %w", err)
			}
		}

		return nil, err
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

func (s *Service) SignInWithPassword(ctx context.Context, email, password string) (*User, error) {
	user, err := s.signInWithPassword(ctx, email, password)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	return user, nil
}
