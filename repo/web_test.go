package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/session"
	"github.com/polyscone/tofu/repo"
	"github.com/polyscone/tofu/repo/sqlite"
)

func TestWebSession(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		session.TestManager(t, func() session.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(repo.NewWeb(ctx, db, 10*time.Minute))

			return repo
		})
	})
}
