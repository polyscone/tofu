package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/repo/account"
	"github.com/polyscone/tofu/internal/repo/account/repotest"
)

func TestUserRepo(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(account.NewSQLiteUserRepo(ctx, db, []byte("s")))

	repotest.RunUserTests(t, repo)
}
