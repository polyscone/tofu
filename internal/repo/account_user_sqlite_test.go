package repo_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/repotest"
)

func TestUserRepo(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))

	repotest.RunAccountUserTests(t, repo)
}
