package system_test

import (
	"context"

	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/event"
	"github.com/polyscone/tofu/sqlite"
)

func NewTestEnv(ctx context.Context) (*system.Service, event.Broker, system.ReadWriter) {
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errsx.Must(sqlite.NewSystemRepo(ctx, db))
	svc := errsx.Must(system.NewService(broker, repo))

	broker.Synchronous = true

	return svc, broker, repo
}
