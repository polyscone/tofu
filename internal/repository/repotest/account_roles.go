package repotest

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/repository"
	"golang.org/x/exp/slices"
)

func AccountRoles(ctx context.Context, t *testing.T, newRepo func() account.ReadWriter) {
	t.Run("add and find", func(t *testing.T) {
		repo := newRepo()

		tt := []struct {
			name string
			role account.Role
			want error
		}{
			{"no data", account.Role{}, nil},
			{"minimal data", account.Role{Name: "Name 1"}, nil},
			{"maximal data", account.Role{
				Name:        "With description",
				Description: "This is a basic description.",
				Permissions: []string{"1", "2", "3"},
			}, nil},
			{"conflicting name", account.Role{Name: "Name 1"}, repository.ErrConflict},
			{"conflicting name with different casing", account.Role{Name: "NAME 1"}, repository.ErrConflict},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := repo.AddRole(ctx, &tc.role)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				found, err := repo.FindRoleByID(ctx, tc.role.ID)
				if err != nil {
					switch {
					case tc.want == nil:
						t.Fatal(err)

					case !errors.Is(err, repository.ErrNotFound):
						t.Errorf("want error: %v; got %v", repository.ErrNotFound, err)
					}
				}

				if tc.want != nil {
					return
				}

				accountRolesEqual(t, &tc.role, found)
			})
		}
	})

	t.Run("add and remove", func(t *testing.T) {
		repo := newRepo()

		role := &account.Role{Name: "Role name"}
		if err := repo.AddRole(ctx, role); err != nil {
			t.Fatal(err)
		}

		if err := repo.RemoveRole(ctx, role.ID); err != nil {
			t.Fatalf("want error: <nil>; got %v", err)
		}

		if _, err := repo.FindRoleByID(ctx, role.ID); err == nil {
			t.Errorf("want error: %v; got <nil>", repository.ErrNotFound)
		}
	})

	t.Run("generate new id on add", func(t *testing.T) {
		repo := newRepo()

		role1 := &account.Role{Name: "Role 1"}
		role2 := &account.Role{Name: "Role 2"}

		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2.ID = role1.ID

		if err := repo.AddRole(ctx, role2); err != nil {
			t.Fatal(err)
		}

		if role2.ID == 0 || role2.ID == role1.ID {
			t.Error("want new id to be generated on add")
		}
	})

	t.Run("save and find", func(t *testing.T) {
		repo := newRepo()

		tt := []struct {
			name string
			role account.Role
			want error
		}{
			{"no data", account.Role{}, nil},
			{"minimal data", account.Role{Name: "Save name 1"}, nil},
			{"conflicting name", account.Role{Name: "Save name 1"}, repository.ErrConflict},
			{"conflicting name with different casing", account.Role{Name: "SAVE NAME 1"}, repository.ErrConflict},
			{"with permissions", account.Role{Name: "Save name 2", Permissions: []string{"1", "2", "3"}}, nil},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				role := account.Role{Name: "New role " + strconv.Itoa(i)}
				if err := repo.AddRole(ctx, &role); err != nil {
					t.Fatal(err)
				}

				if tc.role.ID == 0 {
					tc.role.ID = role.ID
				}

				err := repo.SaveRole(ctx, &tc.role)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				if tc.want != nil {
					return
				}

				found, err := repo.FindRoleByID(ctx, tc.role.ID)
				if err != nil {
					t.Fatal(err)
				}

				accountRolesEqual(t, &tc.role, found)
			})
		}
	})

	t.Run("conflicts", func(t *testing.T) {
		repo := newRepo()

		role1 := &account.Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2 := &account.Role{Name: "Bar"}
		if err := repo.AddRole(ctx, role2); err != nil {
			t.Fatal(err)
		}
		role2.Name = role1.Name

		tt := []struct {
			name string
			err  error
		}{
			{"add", repo.AddRole(ctx, role1)},
			{"save", repo.SaveRole(ctx, role2)},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				var conflict *repository.ConflictError
				if !errors.As(tc.err, &conflict) {
					t.Fatalf("want %T; got %T", conflict, tc.err)
				}

				if conflict.Get("name") == "" {
					t.Error("want name conflict error; got empty string")
				}
			})
		}
	})
}

func accountRolesEqual(t *testing.T, want, got *account.Role) {
	t.Helper()

	if want, got := want.ID, got.ID; want != got {
		t.Errorf("want id to be %v; got %v", want, got)
	}
	if want, got := want.Name, got.Name; want != got {
		t.Errorf("want name to be %v; got %v", want, got)
	}
	if want, got := want.Description, got.Description; want != got {
		t.Errorf("want description to be %v; got %v", want, got)
	}
	if want, got := want.Permissions, got.Permissions; len(want) != len(got) {
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
}
