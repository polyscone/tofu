package sqlite_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo/repotest"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

func TestAccount(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(sqlite.NewAccountRepo(ctx, db))

	t.Run("sqlite", func(t *testing.T) {
		repotest.Account(ctx, t, repo)
	})
}
