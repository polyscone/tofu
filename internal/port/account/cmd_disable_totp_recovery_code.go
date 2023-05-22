package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type disableTOTPRecoveryCodeRequest struct {
	userID       uuid.V4
	recoveryCode RecoveryCode
}

type DisableTOTPWithRecoveryCode struct {
	Guard        DisableTOTPGuard
	UserID       string
	RecoveryCode string
}

func (cmd DisableTOTPWithRecoveryCode) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd DisableTOTPWithRecoveryCode) Validate() error {
	_, err := cmd.request()

	return errors.Tracef(err)
}

func (cmd DisableTOTPWithRecoveryCode) request() (disableTOTPRecoveryCodeRequest, error) {
	var req disableTOTPRecoveryCodeRequest
	if !cmd.Guard.CanDisableTOTP(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.recoveryCode, err = NewRecoveryCode(cmd.RecoveryCode); err != nil {
		errs.Set("recovery code", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type DisableTOTPRecoveryCodeHandler func(ctx context.Context, cmd DisableTOTPWithRecoveryCode) error

func NewDisableTOTPRecoveryCodeHandler(broker event.Broker, users UserRepo) DisableTOTPRecoveryCodeHandler {
	return func(ctx context.Context, cmd DisableTOTPWithRecoveryCode) error {
		req, err := cmd.request()
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.DisableTOTPWithRecoveryCode(req.recoveryCode); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
