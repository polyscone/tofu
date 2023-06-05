package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type createRoleGuard struct {
	value bool
}

func (g createRoleGuard) CanCreateRoles() bool {
	return g.value
}

func TestCreateRole(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	validGuard := createRoleGuard{value: true}
	invalidGuard := createRoleGuard{value: false}

	tt := []struct {
		name     string
		guard    createRoleGuard
		roleName string
		want     error
	}{
		{"unauthorised", invalidGuard, "", app.ErrUnauthorised},
		{"empty name", validGuard, "", app.ErrMalformedInput},
		{"whitespace name", validGuard, "     ", app.ErrMalformedInput},
		{"valid name", validGuard, "Role 1", nil},
		{"conflicting name", validGuard, "ROLE 1", app.ErrConflictingInput},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			created, err := svc.CreateRole(ctx, tc.guard, tc.roleName)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("want error: %v; got %v", tc.want, err)
			}

			if tc.want != nil {
				return
			}

			found := errors.Must(store.FindRoleByID(ctx, created.ID))

			if want, got := created.Name, found.Name; want != got {
				t.Errorf("want name to be %q; got %q", want, got)
			}
		})
	}

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(name text.Name) (*account.Role, error) {
			role, err := svc.CreateRole(ctx, validGuard, name.String())

			return role, err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(name text.Name) bool {
				role, err := execute(name)
				if err != nil {
					t.Log(err)

					return false
				}

				found := errors.Must(store.FindRoleByID(ctx, role.ID))

				return found.Name == name.String()
			})
		})

		t.Run("invalid name", func(t *testing.T) {
			quick.Check(t, func(name quick.Invalid[text.Name]) bool {
				_, err := execute(name.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
