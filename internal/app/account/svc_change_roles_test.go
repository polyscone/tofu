package account_test

import (
	"context"
	"sort"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type setRolesGuard struct {
	canChangeRoles     bool
	canAssignSuperRole bool
}

func (g setRolesGuard) CanChangeRoles(userID int) bool {
	return g.canChangeRoles
}

func (g setRolesGuard) CanAssignSuperRole(userID int) bool {
	return g.canAssignSuperRole
}

func TestChangeRoles(t *testing.T) {
	validSuperGuard := setRolesGuard{canChangeRoles: true, canAssignSuperRole: true}
	validGuard := setRolesGuard{canChangeRoles: true}
	invalidGuard := setRolesGuard{}

	t.Run("success with activated signed in user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Activate: true})

		role1 := MustAddRole(t, ctx, store, TestRole{Name: "Role 1", Permissions: []string{"1", "2"}})
		role2 := MustAddRole(t, ctx, store, TestRole{Name: "Role 2", Permissions: []string{"2", "3"}})
		superRole := errors.Must(store.FindRoleByName(ctx, account.SuperRole.Name))

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.ChangeRoles(ctx, validSuperGuard, user.ID, role1.ID, role2.ID, superRole.ID)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.RolesChanged{Email: user.Email})

		user = errors.Must(store.FindUserByID(ctx, user.ID))

		if want, got := []*account.Role{role1, role2, superRole}, user.Roles; len(want) != len(got) {
			t.Errorf("want %v roles; got %v", len(want), len(got))
		} else {
			sort.Slice(want, func(i, j int) bool { return want[i].ID < want[j].ID })
			sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })

			for i, wantRole := range want {
				gotRole := got[i]

				if wantRole.ID != gotRole.ID {
					t.Errorf("want role %q; got %q", wantRole.Name, gotRole.Name)
				}
			}
		}

		// Change roles without removing super
		err = svc.ChangeRoles(ctx, validSuperGuard, user.ID, superRole.ID)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.RolesChanged{Email: user.Email})
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Activate: true})
		super := MustAddUser(t, ctx, store, TestUser{Email: "super@bloggs.com", Activate: true})

		role1 := MustAddRole(t, ctx, store, TestRole{Name: "Role 1", Permissions: []string{"1", "2"}})
		role2 := MustAddRole(t, ctx, store, TestRole{Name: "Role 2", Permissions: []string{"2", "3"}})
		superRole := errors.Must(store.FindRoleByName(ctx, account.SuperRole.Name))

		errors.Must0(svc.ChangeRoles(ctx, validSuperGuard, super.ID, superRole.ID))

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name    string
			guard   setRolesGuard
			userID  int
			roleIDs []int
			want    error
		}{
			{"unauthorised", invalidGuard, 0, nil, app.ErrUnauthorised},
			{"non-existent user id", validGuard, 0, []int{role1.ID, role2.ID}, app.ErrMalformedInput},
			{"non-existent role ids", validGuard, user.ID, []int{-1, 0}, app.ErrMalformedInput},
			{"unauthorised assignment of super role", validGuard, user.ID, []int{superRole.ID}, app.ErrUnauthorised},
			{"removing super role", validGuard, super.ID, nil, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ChangeRoles(ctx, tc.guard, tc.userID, tc.roleIDs...)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
