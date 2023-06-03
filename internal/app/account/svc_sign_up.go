package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo"
)

func (s *Service) SignUp(ctx context.Context, email string) (*User, error) {
	var input struct {
		email text.Email
	}
	{
		var err error
		var errs errors.Map

		if input.email, err = text.NewEmail(email); err != nil {
			errs.Set("email", err)
		}

		if errs != nil {
			return nil, errs.Tracef(app.ErrMalformedInput)
		}
	}

	user := NewUser(input.email)

	if err := user.SignUp(); err != nil {
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
