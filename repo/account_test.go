package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/repo"
	"github.com/polyscone/tofu/repo/sqlite"
)

func TestAccount(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		account.TestRepo(ctx, t, func() account.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(repo.NewAccount(ctx, db, 1*time.Minute))

			return repo
		})
	})
}
