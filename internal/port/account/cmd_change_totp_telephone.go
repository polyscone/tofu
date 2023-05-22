package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type ChangeTOTPTelephoneGuard interface {
	CanChangeTOTPTelephone(userID uuid.V4) bool
}

type changeTOTPTelephoneRequest struct {
	userID       uuid.V4
	newTelephone text.Telephone
}

type ChangeTOTPTelephone struct {
	Guard        ChangeTOTPTelephoneGuard
	UserID       string
	NewTelephone string
}

func (cmd ChangeTOTPTelephone) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd ChangeTOTPTelephone) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd ChangeTOTPTelephone) request() (changeTOTPTelephoneRequest, error) {
	var req changeTOTPTelephoneRequest
	if !cmd.Guard.CanChangeTOTPTelephone(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.newTelephone, err = text.NewTelephone(cmd.NewTelephone); err != nil {
		errs.Set("new telephone", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type ChangeTOTPTelephoneHandler func(ctx context.Context, cmd ChangeTOTPTelephone) error

func NewChangeTOTPTelephoneHandler(broker event.Broker, users UserRepo) ChangeTOTPTelephoneHandler {
	return func(ctx context.Context, cmd ChangeTOTPTelephone) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.ChangeTOTPTelephone(req.newTelephone); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
