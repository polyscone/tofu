package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo"
)

func (s *Service) SignUp(ctx context.Context, email, password, passwordCheck string) (*User, error) {
	var input struct {
		email         text.Email
		password      Password
		passwordCheck Password
	}
	{
		var err error
		var errs errors.Map

		input.passwordCheck, _ = NewPassword(passwordCheck)

		if input.email, err = text.NewEmail(email); err != nil {
			errs.Set("email", err)
		}
		if input.password, err = NewPassword(password); err != nil {
			errs.Set("password", err)
		} else if !input.password.Equal(input.passwordCheck) {
			errs.Set("password", "passwords do not match")
		}

		if errs != nil {
			return nil, errs.Tracef(app.ErrMalformedInput)
		}
	}

	user := NewUser(input.email)

	if err := user.SignUpWithPassword(input.password, s.hasher); err != nil {
		return nil, errors.Tracef(err)
	}

	if err := s.repo.AddUser(ctx, user); err != nil {
		if errors.Is(err, repo.ErrConflict) {
			errs := errors.Map{"email": errors.New("already in use")}

			return nil, errs.Tracef(app.ErrConflictingInput)
		}

		return nil, errors.Tracef(err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
