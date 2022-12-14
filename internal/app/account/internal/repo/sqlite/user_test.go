package sqlite_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/account/internal/repo/sqlite/repotest"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

func TestUser(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(account.NewSQLiteUserRepo(ctx, db))

	repotest.RunUserTests(t, repo)
}
