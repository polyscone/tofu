package sqlite_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo/repotest"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

func TestAccount(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		repotest.Account(ctx, t, func() account.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errors.Must(sqlite.NewAccountRepo(ctx, db))

			return repo
		})
	})
}
