package repotest

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/system"
)

func System(ctx context.Context, t *testing.T, newRepo func() system.ReadWriter) {
	t.Run("config", func(t *testing.T) { SystemConfig(ctx, t, newRepo) })
}
