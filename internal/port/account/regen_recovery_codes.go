package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

type RegenerateRecoveryCodesGuard interface {
	CanRegenerateRecoveryCodes(userID uuid.V4) bool
}

type regenerateRecoveryCodesRequest struct {
	userID uuid.V4
}

type regenerateRecoveryCodesResponse struct {
	RecoveryCodes []string
}

type RegenerateRecoveryCodes struct {
	Guard  RegenerateRecoveryCodesGuard
	UserID string
}

func (cmd RegenerateRecoveryCodes) Execute(ctx context.Context, bus command.Bus) (regenerateRecoveryCodesResponse, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(regenerateRecoveryCodesResponse), errors.Tracef(err)
}

func (cmd RegenerateRecoveryCodes) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd RegenerateRecoveryCodes) request(ctx context.Context) (regenerateRecoveryCodesRequest, error) {
	var req regenerateRecoveryCodesRequest
	if !cmd.Guard.CanRegenerateRecoveryCodes(uuid.ParseV4OrNil(cmd.UserID)) {
		return req, errors.Tracef(port.ErrUnauthorised)
	}

	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	return req, errs.Tracef(port.ErrMalformedInput)
}

type RegenerateRecoveryCodesHandler func(ctx context.Context, cmd RegenerateRecoveryCodes) (regenerateRecoveryCodesResponse, error)

func NewRegenerateRecoveryCodesHandler(broker event.Broker, users UserRepo) RegenerateRecoveryCodesHandler {
	return func(ctx context.Context, cmd RegenerateRecoveryCodes) (regenerateRecoveryCodesResponse, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return regenerateRecoveryCodesResponse{}, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return regenerateRecoveryCodesResponse{}, errors.Tracef(err)
		}

		err = user.RegenerateRecoveryCodes()
		if err != nil {
			return regenerateRecoveryCodesResponse{}, errors.Tracef(err)
		}

		if err := users.Save(ctx, user); err != nil {
			return regenerateRecoveryCodesResponse{}, errors.Tracef(err)
		}

		broker.Flush(&user.Events)

		recoveryCodes := make([]string, len(user.RecoveryCodes))
		for i, code := range user.RecoveryCodes {
			recoveryCodes[i] = code.String()
		}

		res := regenerateRecoveryCodesResponse{
			RecoveryCodes: recoveryCodes,
		}

		return res, nil
	}
}
