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

type authenticateWithRecoveryCodeRequest struct {
	userID       uuid.V4
	recoveryCode domain.RecoveryCode
}

type AuthenticateWithRecoveryCode struct {
	UserID       string
	RecoveryCode string
}

func (cmd AuthenticateWithRecoveryCode) Execute(ctx context.Context, bus command.Bus) error {
	_, err := bus.Dispatch(ctx, cmd)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithRecoveryCode) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithRecoveryCode) request(ctx context.Context) (authenticateWithRecoveryCodeRequest, error) {
	var req authenticateWithRecoveryCodeRequest
	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}
	if req.recoveryCode, err = domain.NewRecoveryCode(cmd.RecoveryCode); err != nil {
		errs.Set("recovery code", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type AuthenticateWithRecoveryCodeHandler func(ctx context.Context, cmd AuthenticateWithRecoveryCode) error

func NewAuthenticateWithRecoveryCodeHandler(broker event.Broker, users UserRepo) AuthenticateWithRecoveryCodeHandler {
	return func(ctx context.Context, cmd AuthenticateWithRecoveryCode) error {
		req, err := cmd.request(ctx)
		if err != nil {
			return errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return errors.Tracef(err)
		}

		if err := user.AuthenticateWithRecoveryCode(req.recoveryCode); err != nil {
			return errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		return nil
	}
}
