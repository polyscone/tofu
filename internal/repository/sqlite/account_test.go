package sqlite_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/repository/repotest"
	"github.com/polyscone/tofu/internal/repository/sqlite"
)

func TestAccount(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		repotest.Account(ctx, t, func() account.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(sqlite.NewAccountRepo(ctx, db))

			return repo
		})
	})
}
