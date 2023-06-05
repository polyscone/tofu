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
		name  string
		guard createRoleGuard
		role  account.Role
		want  error
	}{
		{"unauthorised", invalidGuard, account.Role{Name: ""}, app.ErrUnauthorised},
		{"empty name", validGuard, account.Role{Name: ""}, app.ErrMalformedInput},
		{"whitespace name", validGuard, account.Role{Name: "     "}, app.ErrMalformedInput},
		{"valid name", validGuard, account.Role{Name: "Role 1"}, nil},
		{"valid description", validGuard, account.Role{Name: "Role 2", Description: "Role description"}, nil},
		{"conflicting name", validGuard, account.Role{Name: "ROLE 1"}, app.ErrConflictingInput},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			created, err := svc.CreateRole(ctx, tc.guard, tc.role.Name, tc.role.Description)
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

		execute := func(name text.Name, description text.OptionalDesc) (*account.Role, error) {
			role, err := svc.CreateRole(ctx, validGuard, name.String(), description.String())

			return role, err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(name text.Name, descrpition text.OptionalDesc) bool {
				_, err := execute(name, descrpition)
				if err != nil {
					t.Log(err)

					return false
				}

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid name", func(t *testing.T) {
			quick.Check(t, func(name quick.Invalid[text.Name], description text.OptionalDesc) bool {
				_, err := execute(name.Unwrap(), description)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid description", func(t *testing.T) {
			quick.Check(t, func(name text.Name, description quick.Invalid[text.OptionalDesc]) bool {
				_, err := execute(name, description.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
