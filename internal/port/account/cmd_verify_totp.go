package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type VerifyTOTPGuard interface {
	CanVerifyTOTP(userID uuid.V4) bool
}

type verifyTOTPRequest struct {
	userID uuid.V4
	totp   TOTP
	useSMS bool
}

type VerifyTOTP struct {
	Guard  VerifyTOTPGuard
	UserID string
	TOTP   string
	UseSMS bool
}

func (cmd VerifyTOTP) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd VerifyTOTP) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd VerifyTOTP) request() (verifyTOTPRequest, error) {
	var req verifyTOTPRequest
	if !cmd.Guard.CanVerifyTOTP(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.totp, err = NewTOTP(cmd.TOTP); err != nil {
		errs.Set("totp", err)
	}

	req.useSMS = cmd.UseSMS

	return req, errs.Tracef(port.ErrMalformedInput)
}

type VerifyTOTPHandler func(ctx context.Context, cmd VerifyTOTP) error

func NewVerifyTOTPHandler(broker event.Broker, users UserRepo) VerifyTOTPHandler {
	return func(ctx context.Context, cmd VerifyTOTP) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		var kind string
		if req.useSMS {
			kind = TOTPKindSMS
		} else {
			kind = TOTPKindApp
		}
		if err := user.VerifyTOTP(req.totp, kind); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
