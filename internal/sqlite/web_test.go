package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/sqlite"
)

func TestWebSession(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		session.TestManager(t, func() session.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(sqlite.NewWebRepo(ctx, db, 10*time.Minute))

			return repo
		})
	})
}
