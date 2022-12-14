package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type authenticateWithPasswordData struct {
	email    text.Email
	password domain.Password
}

type AuthenticateWithPassword struct {
	Email    string
	Password string
}

func (cmd AuthenticateWithPassword) Execute(ctx context.Context, bus command.Bus) (Passport, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(Passport), errors.Tracef(err)
}

func (cmd AuthenticateWithPassword) Validate(ctx context.Context) error {
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithPassword) data(ctx context.Context) (authenticateWithPasswordData, error) {
	var data authenticateWithPasswordData
	var err error
	var errs errors.Map

	if data.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}
	if data.password, err = domain.NewPassword(cmd.Password); err != nil {
		errs.Set("password", err)
	}

	return data, errs.Tracef(app.ErrInvalidInput)
}

type AuthenticateWithPasswordHandler func(ctx context.Context, cmd AuthenticateWithPassword) (Passport, error)

func NewAuthenticateWithPasswordHandler(broker event.Broker, users UserRepo) AuthenticateWithPasswordHandler {
	return func(ctx context.Context, cmd AuthenticateWithPassword) (Passport, error) {
		data, err := cmd.data(ctx)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, data.email)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		if err := user.AuthenticateWithPassword(data.password); err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return newPassport(user), nil
	}
}
