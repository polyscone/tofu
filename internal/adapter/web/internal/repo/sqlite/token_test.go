package sqlite_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/adapter/web/internal/repo/repotest"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

func TestTokenRepo(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(web.NewSQLiteTokenRepo(ctx, db))

	repotest.RunTokenTests(t, repo)
}
