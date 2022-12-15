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

type findUserByIDData struct {
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
	_, err := cmd.data(ctx)

	return errors.Tracef(err)
}

func (cmd IssuePassport) data(ctx context.Context) (findUserByIDData, error) {
	var data findUserByIDData
	var err error
	var errs errors.Map

	if data.userID, err = uuid.ParseV4(cmd.UserID); err != nil {
		errs.Set("user id", err)
	}

	data.isAwaitingMFA = cmd.IsAwaitingMFA
	data.isLoggedIn = cmd.IsLoggedIn

	if data.isAwaitingMFA && data.isLoggedIn {
		errs.Set("is awaiting MFA", errors.Tracef("cannot be both logged in an awaiting MFA"))
		errs.Set("is logged in", errors.Tracef("cannot be both logged in an awaiting MFA"))
	}

	return data, errs.Tracef(port.ErrInvalidInput)
}

type IssuePassportHandler func(ctx context.Context, cmd IssuePassport) (Passport, error)

func NewIssuePassportHandler(broker event.Broker, users UserRepo) IssuePassportHandler {
	return func(ctx context.Context, cmd IssuePassport) (Passport, error) {
		data, err := cmd.data(ctx)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		user, err := users.FindByID(ctx, data.userID)
		if err != nil {
			return EmptyPassport, errors.Tracef(err)
		}

		switch {
		case data.isAwaitingMFA:
			user.SetAuthStatus(domain.AwaitingMFA)

		case data.isLoggedIn:
			user.SetAuthStatus(domain.Authenticated)
		}

		return newPassport(user), nil
	}
}
