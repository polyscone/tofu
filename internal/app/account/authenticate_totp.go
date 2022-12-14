package account

import (
	"context"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
)

type authenticateWithTOTPData struct {
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
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd AuthenticateWithTOTP) data(ctx context.Context) (authenticateWithTOTPData, error) {
	var data authenticateWithTOTPData
	var err error
	var errs errors.Map

	data.user = cmd.Passport.user

	if data.totp, err = domain.NewTOTP(cmd.TOTP); err != nil {
		errs.Set("totp", err)
	}

	return data, errs.Tracef(app.ErrInvalidInput)
}

type AuthenticateWithTOTPHandler func(ctx context.Context, cmd AuthenticateWithTOTP) (Passport, error)

func NewAuthenticateWithTOTPHandler(broker event.Broker, users UserRepo) AuthenticateWithTOTPHandler {
	return func(ctx context.Context, cmd AuthenticateWithTOTP) (Passport, error) {
		data, err := cmd.data(ctx)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		if err := data.user.AuthenticateWithTOTP(data.totp); err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		broker.Flush(&data.user.Events)

		return newPassport(data.user), nil
	}
}
