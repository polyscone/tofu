package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/domain"
)

type DisableTOTPGuard interface {
	CanDisableTOTP(userID uuid.V4) bool
}

type disableTOTPRequest struct {
	userID uuid.V4
	totp   domain.TOTP
}

type DisableTOTP struct {
	Guard  DisableTOTPGuard
	UserID string
	TOTP   string
}

func (cmd DisableTOTP) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd DisableTOTP) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd DisableTOTP) request() (disableTOTPRequest, error) {
	var req disableTOTPRequest
	if !cmd.Guard.CanDisableTOTP(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.totp, err = domain.NewTOTP(cmd.TOTP); err != nil {
		errs.Set("totp", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type DisableTOTPHandler func(ctx context.Context, cmd DisableTOTP) error

func NewDisableTOTPHandler(broker event.Broker, users UserRepo) DisableTOTPHandler {
	return func(ctx context.Context, cmd DisableTOTP) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.DisableTOTP(req.totp); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
