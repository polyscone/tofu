package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/repotest"
)

func TestWebSessionRepo(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(repo.NewSQLiteWebSessionRepo(ctx, db, 1*time.Minute))

	repotest.RunWebSessionTests(t, repo)
}
