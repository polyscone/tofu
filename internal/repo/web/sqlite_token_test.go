package web_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/repo/web"
	"github.com/polyscone/tofu/internal/repo/web/repotest"
)

func TestWebTokenRepo(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(web.NewSQLiteTokenRepo(ctx, db))

	repotest.RunTokenTests(t, repo)
}
