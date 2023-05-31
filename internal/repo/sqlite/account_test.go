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

	repotest.Account(ctx, t, repo)
}
