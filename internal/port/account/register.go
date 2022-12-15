package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

type registerData struct {
	userID uuid.V4
	email  text.Email
}

type Register struct {
	UserID string
	Email  string
}

func (cmd Register) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return err
}

func (cmd Register) Validate(ctx context.Context) error {
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd Register) data(ctx context.Context) (registerData, error) {
	var data registerData
	var err error
	var errs errors.Map

	if data.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("id", err)
	}
	if data.email, err = text.NewEmail(cmd.Email); err != nil {
		errs.Set("email", err)
	}

	return data, errs.Tracef(port.ErrInvalidInput)
}

type RegisterHandler func(ctx context.Context, cmd Register) error

func NewRegisterHandler(broker event.Broker, users UserRepo) RegisterHandler {
	return func(ctx context.Context, cmd Register) error {
		data, err := cmd.data(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user := domain.NewUser(data.userID)

		user.Register(data.email)

		if err := users.Add(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
