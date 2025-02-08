package repo_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/repo"
	"github.com/polyscone/tofu/repo/sqlite"
)

func TestSystem(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		ctx := context.Background()

		system.TestRepo(ctx, t, func() system.ReadWriter {
			db := sqlite.OpenInMemoryTestDatabase(ctx)
			repo := errsx.Must(repo.NewSystem(ctx, db))

			return repo
		})
	})
}
