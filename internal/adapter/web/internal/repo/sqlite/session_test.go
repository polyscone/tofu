package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web"
	"github.com/polyscone/tofu/internal/adapter/web/internal/repo/repotest"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
)

func TestSession(t *testing.T) {
	ctx := context.Background()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(web.NewSQLiteSessionRepo(ctx, db, 1*time.Minute))

	repotest.RunSessionTests(t, repo)
}
