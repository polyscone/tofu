package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
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

func (s *Service) SignInWithPassword(ctx context.Context, email, password string) error {
	var input struct {
		email    Email
		password Password
	}
	{
		var err error
		var errs errsx.Map

		if input.email, err = NewEmail(email); err != nil {
			errs.Set("email", err)
		}
		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		}

		if errs != nil {
			return fmt.Errorf("%w: %w", app.ErrMalformedInput, errs)
		}
	}

	log, err := s.repo.FindSignInAttemptLogByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find sign in attempt log by email: %w", err)
	}

	if err := s.CheckSignInThrottle(log.Attempts, log.LastAttemptAt); err != nil {
		return fmt.Errorf("check sign in throttle: %w", err)
	}

	// Even in the case where a user doesn't exist we always want to log
	// failed sign in attempts to prevent leaking information about which
	// users exist in the system
	//
	// These values are set to their zero values on a successful
	// sign in or account verification
	log.Attempts++
	log.LastAttemptAt = time.Now()

	user, err := s.repo.FindUserByEmail(ctx, input.email.String())
	if err != nil {
		if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
			return fmt.Errorf("save sign in attempt log: %w", err)
		}

		// Always check a password even when we error finding a user to help
		// avoid leaking info that would allow enumeration of valid emails
		if err := s.hasher.CheckDummyPasswordHash(); err != nil {
			return fmt.Errorf("check dummy password hash: %w", err)
		}

		return fmt.Errorf("find user by email: %w", err)
	}

	if _, err := user.SignInWithPassword(s.system, input.password, s.hasher); err != nil {
		switch {
		case errors.Is(err, ErrNotVerified),
			errors.Is(err, ErrNotActivated),
			errors.Is(err, ErrInvalidPassword):

			if err := s.repo.SaveSignInAttemptLog(ctx, log); err != nil {
				return fmt.Errorf("save sign in attempt log: %w", err)
			}

			if err := s.repo.SaveUser(ctx, user); err != nil {
				return fmt.Errorf("save user: %w", err)
			}
		}

		return err
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
