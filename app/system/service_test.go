package system_test

import (
	"context"

	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/event"
	"github.com/polyscone/tofu/repo"
	"github.com/polyscone/tofu/repo/sqlite"
)

func NewTestEnv(ctx context.Context) (*system.Service, event.Broker, system.ReadWriter) {
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errsx.Must(repo.NewSystem(ctx, db))
	svc := errsx.Must(system.NewService(broker, repo))

	return svc, broker, repo
}
