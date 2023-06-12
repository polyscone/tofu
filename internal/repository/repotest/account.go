package repotest

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
)

func Account(ctx context.Context, t *testing.T, newRepo func() account.ReadWriter) {
	t.Run("roles", func(t *testing.T) { AccountRoles(ctx, t, newRepo) })
	t.Run("users", func(t *testing.T) { AccountUsers(ctx, t, newRepo) })
}
