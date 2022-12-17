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

type findUserByIDRequest struct {
	userID         uuid.V4
	isAwaitingTOTP bool
	isLoggedIn     bool
}

type IssuePassport struct {
	UserID         string
	IsAwaitingTOTP bool
	IsLoggedIn     bool
}

func (cmd IssuePassport) Execute(ctx context.Context, bus command.Bus) (Passport, error) {
	res, err := bus.Dispatch(ctx, cmd)

	return res.(Passport), errors.Tracef(err)
}

func (cmd IssuePassport) Validate(ctx context.Context) error {
	_, err := cmd.request(ctx)

	return errors.Tracef(err)
}

func (cmd IssuePassport) request(ctx context.Context) (findUserByIDRequest, error) {
	var req findUserByIDRequest
	var err error
	var errs errors.Map

	if req.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	req.isAwaitingTOTP = cmd.IsAwaitingTOTP
	req.isLoggedIn = cmd.IsLoggedIn

	if req.isAwaitingTOTP && req.isLoggedIn {
		errs.Set("is awaiting TOTP", errors.Tracef("cannot be both logged in an awaiting TOTP"))
		errs.Set("is logged in", errors.Tracef("cannot be both logged in an awaiting TOTP"))
	}

	return req, errs.Tracef(port.ErrInvalidInput)
}

type IssuePassportHandler func(ctx context.Context, cmd IssuePassport) (Passport, error)

func NewIssuePassportHandler(broker event.Broker, users UserRepo) IssuePassportHandler {
	return func(ctx context.Context, cmd IssuePassport) (Passport, error) {
		req, err := cmd.request(ctx)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, req.userID)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		switch {
		case req.isAwaitingTOTP:
			user.SetAuthStatus(domain.AwaitingTOTP)

		case req.isLoggedIn:
			user.SetAuthStatus(domain.Authenticated)
		}

		return newPassport(user), nil
	}
}
