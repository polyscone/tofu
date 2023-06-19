package sqlite_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository/repotest"
	"github.com/polyscone/tofu/internal/repository/sqlite"
)

func TestSystem(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		repotest.System(ctx, t, func() system.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(sqlite.NewSystemRepo(ctx, db))

			return repo
		})
	})
}
