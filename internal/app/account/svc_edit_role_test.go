package account_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type updateRoleGuard struct {
	value bool
}

func (g updateRoleGuard) CanCreateRoles() bool {
	return true
}

func (g updateRoleGuard) CanUpdateRoles() bool {
	return g.value
}

func TestUpdateRole(t *testing.T) {
	validGuard := updateRoleGuard{value: true}
	invalidGuard := updateRoleGuard{value: false}

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  updateRoleGuard
			before account.Role
			after  account.Role
			want   error
		}{
			{"authorised", validGuard, account.Role{Name: "Foo"}, account.Role{Name: "Bar"}, nil},
			{"unauthorised", invalidGuard, account.Role{Name: "Foo"}, account.Role{Name: "Bar"}, app.ErrUnauthorised},
			{"conflicting name", validGuard, account.Role{Name: "Qux"}, account.Role{Name: "Bar"}, app.ErrConflictingInput},
			{"empty name", validGuard, account.Role{Name: "Quxx"}, account.Role{Name: ""}, app.ErrMalformedInput},
			{"empty permission update", validGuard, account.Role{
				Name:        "Role 1",
				Description: "Role 1 description",
				Permissions: []string{"1", "2"},
			}, account.Role{
				Name:        "Role 2",
				Description: "Role 2 description",
				Permissions: []string{"3", "", "6"},
			}, app.ErrMalformedInput},
			{"successful update", validGuard, account.Role{
				Name:        "Role 3",
				Description: "Role 3 description",
				Permissions: []string{"1", "2"},
			}, account.Role{
				Name:        "Role 4",
				Description: "Role 4 description",
				Permissions: []string{"3", "1", "6"},
			}, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				before, err := svc.CreateRole(ctx, tc.guard, tc.before.Name, tc.before.Description, tc.before.Permissions)
				if err != nil {
					t.Fatal(err)
				}

				edited, err := svc.UpdateRole(ctx, tc.guard, before.ID, tc.after.Name, tc.after.Description, tc.after.Permissions)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				if tc.want != nil {
					return
				}

				after, err := repo.FindRoleByID(ctx, edited.ID)
				if err != nil {
					t.Fatal(err)
				}

				if want, got := tc.after.Name, after.Name; want != got {
					t.Errorf("want name to be %q; got %q", want, got)
				}
				if want, got := tc.after.Description, after.Description; want != got {
					t.Errorf("want description to be %q; got %q", want, got)
				}
				if want, got := tc.after.Permissions, after.Permissions; len(want) != len(got) {
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
}
