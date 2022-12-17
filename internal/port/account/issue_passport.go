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
	userID        uuid.V4
	isAwaitingMFA bool
	isLoggedIn    bool
}

type IssuePassport struct {
	UserID        string
	IsAwaitingMFA bool
	IsLoggedIn    bool
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

	req.isAwaitingMFA = cmd.IsAwaitingMFA
	req.isLoggedIn = cmd.IsLoggedIn

	if req.isAwaitingMFA && req.isLoggedIn {
		errs.Set("is awaiting MFA", errors.Tracef("cannot be both logged in an awaiting MFA"))
		errs.Set("is logged in", errors.Tracef("cannot be both logged in an awaiting MFA"))
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
		case req.isAwaitingMFA:
			user.SetAuthStatus(domain.AwaitingMFA)

		case req.isLoggedIn:
			user.SetAuthStatus(domain.Authenticated)
		}

		return newPassport(user), nil
	}
}
