package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

type authenticateWithTOTPRequest struct {
	userID uuid.V4
	totp   domain.TOTP
}

type AuthenticateWithTOTP struct {
	UserID string
	TOTP   string
}

func (cmd AuthenticateWithTOTP) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithTOTP) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithTOTP) request(ctx context.Context) (authenticateWithTOTPRequest, error) {
	var req authenticateWithTOTPRequest
	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.totp, err = domain.NewTOTP(cmd.TOTP); err != nil {
		errs.Set("totp", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type AuthenticateWithTOTPHandler func(ctx context.Context, cmd AuthenticateWithTOTP) error

func NewAuthenticateWithTOTPHandler(broker event.Broker, users UserRepo) AuthenticateWithTOTPHandler {
	return func(ctx context.Context, cmd AuthenticateWithTOTP) error {
		req, err := cmd.request(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.AuthenticateWithTOTP(req.totp); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
