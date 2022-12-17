package account

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

type authenticateWithTOTPRequest struct {
	user domain.User
	totp domain.TOTP
}

type AuthenticateWithTOTP struct {
	Passport Passport
	TOTP     string
}

func (cmd AuthenticateWithTOTP) Execute(ctx context.Context, bus command.Bus) (Passport, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(Passport), errors.Tracef(err)
}

func (cmd AuthenticateWithTOTP) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithTOTP) request(ctx context.Context) (authenticateWithTOTPRequest, error) {
	var req authenticateWithTOTPRequest
	var err error
	var errs errors.Map

	req.user = cmd.Passport.user

	if req.totp, err = domain.NewTOTP(cmd.TOTP); err != nil {
		errs.Set("totp", err)
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type AuthenticateWithTOTPHandler func(ctx context.Context, cmd AuthenticateWithTOTP) (Passport, error)

func NewAuthenticateWithTOTPHandler(broker event.Broker, users UserRepo) AuthenticateWithTOTPHandler {
	return func(ctx context.Context, cmd AuthenticateWithTOTP) (Passport, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		if err := req.user.AuthenticateWithTOTP(req.totp); err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		broker.Flush(&req.user.Events)

		return newPassport(req.user), nil
	}
}
