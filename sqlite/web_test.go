package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/session"
	"github.com/polyscone/tofu/sqlite"
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
