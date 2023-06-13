package account_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
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
		svc, broker, repo := NewTestEnv(ctx)

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

				found := errsx.Must(repo.FindRoleByID(ctx, created.ID))

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

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
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
