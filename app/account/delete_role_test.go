package account_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/pkg/errsx"
	"github.com/polyscone/tofu/pkg/testutil"
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
	validGuard := deleteRoleGuard{value: true}
	invalidGuard := deleteRoleGuard{value: false}

	t.Run("with guards", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name  string
			guard deleteRoleGuard
			want  error
		}{
			{"authorised", validGuard, nil},
			{"invalid guard", invalidGuard, app.ErrForbidden},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				roleID := errsx.Must(repo.NextRoleID(ctx))
				name := "Role " + strconv.Itoa(i)
				err := svc.CreateRole(ctx, tc.guard, roleID.String(), name, "", nil)
				if err != nil {
					t.Fatal(err)
				}

				err = svc.DeleteRole(ctx, tc.guard, roleID.String())
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				if tc.want != nil {
					return
				}

				if _, err := repo.FindRoleByID(ctx, roleID.String()); err == nil {
					t.Errorf("want error: %v; got <nil>", app.ErrNotFound)
				}
			})
		}
	})
}