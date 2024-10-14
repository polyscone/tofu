package account_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
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

	t.Run("with guards", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name  string
			guard createRoleGuard
			role  account.Role
			want  error
		}{
			{"invalid guard", invalidGuard, account.Role{Name: ""}, app.ErrForbidden},
			{"empty name", validGuard, account.Role{Name: ""}, app.ErrMalformedInput},
			{"whitespace name", validGuard, account.Role{Name: "     "}, app.ErrMalformedInput},
			{"valid name", validGuard, account.Role{Name: "Role 1"}, nil},
			{"valid description", validGuard, account.Role{Name: "Role 2", Description: "Role description"}, nil},
			{"conflicting name", validGuard, account.Role{Name: "ROLE 1"}, app.ErrConflict},
			{"with permissions", validGuard, account.Role{Name: "Role 3", Permissions: []string{"1", "2", "3"}}, nil},
			{"with empty permission", validGuard, account.Role{Name: "Role 3", Permissions: []string{"1", "", "3"}}, app.ErrMalformedInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				role, err := svc.CreateRole(ctx, tc.guard, tc.role.Name, tc.role.Description, tc.role.Permissions)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				if tc.want != nil {
					return
				}

				found := errsx.Must(repo.FindRoleByID(ctx, role.ID))

				if want, got := tc.role.Name, found.Name; want != got {
					t.Errorf("want name to be %q; got %q", want, got)
				}
				if want, got := tc.role.Description, found.Description; want != got {
					t.Errorf("want description to be %q; got %q", want, got)
				}
				if want, got := tc.role.Permissions, found.Permissions; len(want) != len(got) {
					t.Errorf("want %v permissions; got %v", len(want), len(got))
				} else {
					slices.Sort(want)
					slices.Sort(got)

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

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name         string
			roleName     string
			description  string
			permissions  []string
			isValidInput bool
		}{
			{"valid inputs", "Role name", "Description", []string{"1"}, true},

			{"invalid name empty", "", "Description", []string{"1"}, false},
			{"invalid name whitespace", "   ", "Description", []string{"1"}, false},
			{"invalid name too long", strings.Repeat(".", 31), "Description", []string{"1"}, false},

			{"invalid description too long", "Role name", strings.Repeat(".", 101), []string{"1"}, false},

			{"invalid permission empty", "Role name", "Description", []string{""}, false},
			{"invalid permission whitespace", "Role name", "Description", []string{"   "}, false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.CreateRole(ctx, validGuard, tc.roleName, tc.description, tc.permissions)
				switch {
				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
