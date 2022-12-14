package compose

import (
	"context"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

func Local(ctx context.Context, db *sqlite.DB) (command.Bus, event.Broker, error) {
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
		bus.Register(account.NewIssuePassportHandler(broker, users))
		bus.Register(account.NewRegisterHandler(broker, users))
	}

	return bus, broker, nil
}
