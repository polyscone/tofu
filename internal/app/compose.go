package app

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/port/account"
)

func Compose(ctx context.Context, db *sqlite.DB) (command.Bus, event.Broker, error) {
	bus := command.NewMemoryBus()
	broker := event.NewMemoryBroker()

	// Account
	{
		users, err := account.NewSQLiteUserRepo(ctx, db)
		if err != nil {
			return nil, nil, errors.Tracef(err)
		}

		bus.Register(account.NewActivateHandler(broker, users))
		bus.Register(account.NewAuthenticateWithPasswordHandler(broker, users))
		bus.Register(account.NewAuthenticateWithTOTPHandler(broker, users))
		bus.Register(account.NewChangePasswordHandler(broker, users))
		bus.Register(account.NewFindAuthInfoHandler(broker, users))
		bus.Register(account.NewRegisterHandler(broker, users))
		bus.Register(account.NewSetupTOTPHandler(broker, users))
		bus.Register(account.NewVerifyTOTPHandler(broker, users))
	}

	return bus, broker, nil
}
