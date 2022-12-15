package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

type activateData struct {
	email    text.Email
	password domain.Password
}

type Activate struct {
	Email    string
	Password string
}

func (cmd Activate) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd Activate) Validate(ctx context.Context) error {
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd Activate) data(ctx context.Context) (activateData, error) {
	var data activateData
	var err error
	var errs errors.Map

	if data.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}
	if data.password, err = domain.NewPassword(cmd.Password); err != nil {
		errs.Set("password", err)
	}

	return data, errs.Tracef(port.ErrInvalidInput)
}

type ActivateHandler func(ctx context.Context, cmd Activate) error

func NewActivateHandler(broker event.Broker, users UserRepo) ActivateHandler {
	return func(ctx context.Context, cmd Activate) error {
		data, err := cmd.data(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByEmail(ctx, data.email)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ActivateAndSetPassword(data.password); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
