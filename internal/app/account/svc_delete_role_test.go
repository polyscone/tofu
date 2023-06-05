package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo"
)

type deleteRoleGuard struct {
	value bool
}

func (g deleteRoleGuard) CanCreateRoles() bool {
	return true
}

func (g deleteRoleGuard) CanDeleteRoles() bool {
	return g.value
}

func TestDeleteRole(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	validGuard := deleteRoleGuard{value: true}
	invalidGuard := deleteRoleGuard{value: false}

	events := testutil.NewEventLog(broker)
	defer events.Check(t)

	tt := []struct {
		name  string
		guard deleteRoleGuard
		want  error
	}{
		{"authorised", validGuard, nil},
		{"unauthorised", invalidGuard, app.ErrUnauthorised},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			name := text.GenerateName().String()
			description := text.GenerateDesc().String()
			role, err := svc.CreateRole(ctx, tc.guard, name, description)
			if err != nil {
				t.Fatal(err)
			}

			deleted, err := svc.DeleteRole(ctx, tc.guard, role.ID)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("want error: %v; got %v", tc.want, err)
			}

			if tc.want != nil {
				return
			}

			if _, err := store.FindRoleByID(ctx, deleted.ID); err == nil {
				t.Errorf("want error: %v; got <nil>", repo.ErrNotFound)
			}
		})
	}
}
