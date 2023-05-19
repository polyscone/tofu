package app

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/port/account"
)

func Compose(ctx context.Context, db *sqlite.DB, secret []byte, hasher password.Hasher) (command.Bus, event.Broker, error) {
	bus := command.NewMemoryBus()
	broker := event.NewMemoryBroker()

	// Account
	{
		users, err := account.NewSQLiteUserRepo(ctx, db, secret)
		if err != nil {
			return nil, nil, errors.Tracef(err)
		}

		bus.Register(account.NewActivateHandler(broker, users))
		bus.Register(account.NewAuthenticateWithPasswordHandler(broker, hasher, users))
		bus.Register(account.NewAuthenticateWithRecoveryCodeHandler(broker, users))
		bus.Register(account.NewAuthenticateWithTOTPHandler(broker, users))
		bus.Register(account.NewChangePasswordHandler(broker, hasher, users))
		bus.Register(account.NewChangeTOTPTelephoneHandler(broker, users))
		bus.Register(account.NewDisableTOTPHandler(broker, users))
		bus.Register(account.NewDisableTOTPRecoveryCodeHandler(broker, users))
		bus.Register(account.NewFindUserByEmailHandler(broker, users))
		bus.Register(account.NewFindUserByIDHandler(broker, users))
		bus.Register(account.NewGenerateTOTPHandler(broker, users))
		bus.Register(account.NewRegenerateRecoveryCodesHandler(broker, users))
		bus.Register(account.NewRegisterHandler(broker, hasher, users))
		bus.Register(account.NewResetPasswordHandler(broker, hasher, users))
		bus.Register(account.NewSetupTOTPHandler(broker, users))
		bus.Register(account.NewVerifyTOTPHandler(broker, users))
	}

	return bus, broker, nil
}
