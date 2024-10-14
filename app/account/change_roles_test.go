package account_test

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type setRolesGuard struct {
	canChangeRoles bool
}

func (g setRolesGuard) CanChangeRoles(userID int) bool {
	return g.canChangeRoles
}

func TestChangeRoles(t *testing.T) {
	validGuard := setRolesGuard{canChangeRoles: true}
	invalidGuard := setRolesGuard{}

	t.Run("success", func(t *testing.T) {
		for _, verify := range []bool{true, false} {
			name := "unverified"
			if verify {
				name = "verified"
			}

			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				svc, broker, repo := NewTestEnv(ctx)

				user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Verify: verify})
				role1 := MustAddRole(t, ctx, repo, TestRole{Name: "Role 1", Permissions: []string{"1", "2"}})
				role2 := MustAddRole(t, ctx, repo, TestRole{Name: "Role 2", Permissions: []string{"2", "3"}})

				events := testx.NewEventLog(broker)
				defer events.Check(t)

				roleIDs := []int{role1.ID, role2.ID}
				grants := []string{"a", "b", "c"}
				denials := []string{"b", "c", "d"}
				_, err := svc.ChangeRoles(ctx, validGuard, user.ID, roleIDs, grants, denials)
				if err != nil {
					t.Fatal(err)
				}

				events.Expect(account.RolesChanged{Email: user.Email})

				user = errsx.Must(repo.FindUserByID(ctx, user.ID))

				if want, got := []*account.Role{role1, role2}, user.Roles; len(want) != len(got) {
					t.Errorf("want %v roles; got %v", len(want), len(got))
				} else {
					slices.SortFunc(want, func(a, b *account.Role) int { return cmp.Compare(a.ID, b.ID) })
					slices.SortFunc(got, func(a, b *account.Role) int { return cmp.Compare(a.ID, b.ID) })

					for i, wantRole := range want {
						gotRole := got[i]

						if wantRole.ID != gotRole.ID {
							t.Errorf("want role %q; got %q", wantRole.Name, gotRole.Name)
						}
					}
				}

				if want, got := []string{"a"}, user.Grants; len(want) != len(got) {
					t.Errorf("want %v grants; got %v", len(want), len(got))
				} else {
					slices.Sort(want)
					slices.Sort(got)

					for i, wantGrant := range want {
						gotGrant := got[i]

						if wantGrant != gotGrant {
							t.Errorf("want grant %q; got %q", wantGrant, gotGrant)
						}
					}
				}

				if want, got := []string{"b", "c", "d"}, user.Denials; len(want) != len(got) {
					t.Errorf("want %v denials; got %v", len(want), len(got))
				} else {
					slices.Sort(want)
					slices.Sort(got)

					for i, wantDenial := range want {
						gotDenial := got[i]

						if wantDenial != gotDenial {
							t.Errorf("want denial %q; got %q", wantDenial, gotDenial)
						}
					}
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})
		role1 := MustAddRole(t, ctx, repo, TestRole{Name: "Role 1", Permissions: []string{"1", "2"}})
		role2 := MustAddRole(t, ctx, repo, TestRole{Name: "Role 2", Permissions: []string{"2", "3"}})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name    string
			guard   setRolesGuard
			userID  int
			roleIDs []int
			want    error
		}{
			{"invalid guard", invalidGuard, 0, nil, app.ErrForbidden},
			{"non-existent user id", validGuard, 999, []int{role1.ID, role2.ID}, app.ErrNotFound},
			{"non-existent role ids", validGuard, user.ID, []int{888, 999}, app.ErrNotFound},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.ChangeRoles(ctx, tc.guard, tc.userID, tc.roleIDs, nil, nil)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
