package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/sqlite"
)

func TestAccount(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		account.TestRepo(ctx, t, func() account.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(sqlite.NewAccountRepo(ctx, db, 1*time.Minute))

			return repo
		})
	})
}