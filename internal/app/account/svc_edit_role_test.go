package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type editRoleGuard struct {
	value bool
}

func (g editRoleGuard) CanCreateRoles() bool {
	return true
}

func (g editRoleGuard) CanEditRoles() bool {
	return g.value
}

func TestEditRole(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	validGuard := editRoleGuard{value: true}
	invalidGuard := editRoleGuard{value: false}

	events := testutil.NewEventLog(broker)
	defer events.Check(t)

	tt := []struct {
		name   string
		guard  editRoleGuard
		before *account.Role
		after  *account.Role
		want   error
	}{
		{"authorised", validGuard, &account.Role{Name: "Foo"}, &account.Role{Name: "Bar"}, nil},
		{"unauthorised", invalidGuard, &account.Role{Name: "Foo"}, &account.Role{Name: "Bar"}, app.ErrUnauthorised},
		{"conflicting name", validGuard, &account.Role{Name: "Qux"}, &account.Role{Name: "Bar"}, app.ErrConflictingInput},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			before, err := svc.CreateRole(ctx, tc.guard, tc.before.Name)
			if err != nil {
				t.Fatal(err)
			}

			edited, err := svc.EditRole(ctx, tc.guard, before.ID, tc.after.Name)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("want error: %v; got %v", tc.want, err)
			}

			if tc.want != nil {
				return
			}

			after, err := store.FindRoleByID(ctx, edited.ID)
			if err != nil {
				t.Fatal(err)
			}

			if want, got := tc.after.Name, after.Name; want != got {
				t.Errorf("want name to be %q; got %q", want, got)
			}
		})
	}
}
