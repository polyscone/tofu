package system_test

import (
	"context"

	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

func NewTestEnv(ctx context.Context) (*system.Service, event.Broker, system.ReadWriter) {
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	store := errors.Must(sqlite.NewSystemStore(ctx, db))
	svc := errors.Must(system.NewService(broker, store))

	return svc, broker, store
}
