package account_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
)

type createRoleGuard struct {
	value bool
}

func (g createRoleGuard) CanCreateRoles() bool {
	return g.value
}

func TestCreateRole(t *testing.T) {
	validGuard := createRoleGuard{value: true}
	invalidGuard := createRoleGuard{value: false}

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

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
			{"with permissions", validGuard, account.Role{Name: "Role 3", Permissions: []string{"1", "2", "3"}}, nil},
			{"with empty permission", validGuard, account.Role{Name: "Role 3", Permissions: []string{"1", "", "3"}}, app.ErrMalformedInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				created, err := svc.CreateRole(ctx, tc.guard, tc.role.Name, tc.role.Description, tc.role.Permissions)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				if tc.want != nil {
					return
				}

				found := errsx.Must(store.FindRoleByID(ctx, created.ID))

				if want, got := created.Name, found.Name; want != got {
					t.Errorf("want name to be %q; got %q", want, got)
				}
				if want, got := created.Description, found.Description; want != got {
					t.Errorf("want description to be %q; got %q", want, got)
				}
				if want, got := created.Permissions, found.Permissions; len(want) != len(got) {
					t.Errorf("want %v permissions; got %v", len(want), len(got))
				} else {
					sort.Strings(want)
					sort.Strings(got)

					for i, wantPermission := range want {
						gotPermission := got[i]

						if wantPermission != gotPermission {
							t.Errorf("want permission %q; got %q", wantPermission, gotPermission)
						}
					}
				}
			})
		}
	})

	t.Run("properties", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(name account.RoleName, description account.RoleDesc, permission account.Permission) (*account.Role, error) {
			permissions := []string{permission.String()}
			role, err := svc.CreateRole(ctx, validGuard, name.String(), description.String(), permissions)

			return role, err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(name account.RoleName, descrpition account.RoleDesc, permission account.Permission) bool {
				_, err := execute(name, descrpition, permission)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid name", func(t *testing.T) {
			quick.Check(t, func(name quick.Invalid[account.RoleName], description account.RoleDesc, permission account.Permission) bool {
				_, err := execute(name.Unwrap(), description, permission)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid description", func(t *testing.T) {
			quick.Check(t, func(name account.RoleName, description quick.Invalid[account.RoleDesc], permission account.Permission) bool {
				_, err := execute(name, description.Unwrap(), permission)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid permission", func(t *testing.T) {
			quick.Check(t, func(name account.RoleName, description account.RoleDesc, permission quick.Invalid[account.Permission]) bool {
				_, err := execute(name, description, permission.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
