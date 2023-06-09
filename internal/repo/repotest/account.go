package repotest

import (
	"bytes"
	"context"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/repo"
)

func Account(ctx context.Context, t *testing.T, newStore func() account.ReadWriter) {
	t.Run("users", func(t *testing.T) {
		t.Run("conflicts", func(t *testing.T) {
			store := newStore()

			user1 := &account.User{Email: "Foo"}
			if err := store.AddUser(ctx, user1); err != nil {
				t.Fatal(err)
			}

			user2 := &account.User{Email: "Bar"}
			if err := store.AddUser(ctx, user2); err != nil {
				t.Fatal(err)
			}
			user2.Email = user1.Email

			tt := []struct {
				name string
				err  error
			}{
				{"add", store.AddUser(ctx, user1)},
				{"save", store.SaveUser(ctx, user2)},
			}
			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					var conflicts *repo.ConflictError
					if !errors.As(tc.err, &conflicts) {
						t.Fatalf("want %T; got %T", conflicts, tc.err)
					}

					if conflicts.Get("email") == "" {
						t.Error("want email conflict error; got empty string")
					}
				})
			}
		})

		t.Run("add and find", func(t *testing.T) {
			store := newStore()

			role1 := &account.Role{Name: "Foo"}
			if err := store.AddRole(ctx, role1); err != nil {
				t.Fatal(err)
			}

			role2 := &account.Role{Name: "Bar"}
			if err := store.AddRole(ctx, role2); err != nil {
				t.Fatal(err)
			}

			tt := []struct {
				name string
				user account.User
				want error
			}{
				{"no data", account.User{}, nil},
				{"minimal data", account.User{Email: "Email 1"}, nil},
				{"maximal data", account.User{
					Email:              "Email 2",
					HashedPassword:     []byte("HashedPassword"),
					TOTPMethod:         "TOTPMethod",
					TOTPTel:            "TOTPTel",
					TOTPKey:            []byte("TOTPKey"),
					TOTPAlgorithm:      "TOTPAlgorithm",
					TOTPDigits:         123,
					TOTPPeriod:         456,
					TOTPVerifiedAt:     time.Now(),
					TOTPActivatedAt:    time.Now(),
					SignedUpAt:         time.Now(),
					ActivatedAt:        time.Now(),
					LastSignedInAt:     time.Now(),
					LastSignedInMethod: "Website",
					RecoveryCodes:      []string{"1", "2", "3", "4"},
					Roles:              []*account.Role{role1, role2},
					Grants:             []string{"a", "b"},
					Denials:            []string{"b", "c", "d"},
				}, nil},
				{"conflicting email", account.User{Email: "Email 1"}, repo.ErrConflict},
				{"conflicting email with different casing", account.User{Email: "EMAIL 1"}, repo.ErrConflict},
			}
			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					err := store.AddUser(ctx, &tc.user)
					if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
						t.Fatalf("want error: %v; got %v", tc.want, err)
					}

					found, err := store.FindUserByID(ctx, tc.user.ID)
					if err != nil {
						switch {
						case tc.want == nil:
							t.Fatal(err)

						case !errors.Is(err, repo.ErrNotFound):
							t.Errorf("want error: %v; got %v", repo.ErrNotFound, err)
						}
					}

					if tc.want != nil {
						return
					}

					accountUsersEqual(t, &tc.user, found)
				})
			}
		})

		t.Run("generate new id on add", func(t *testing.T) {
			store := newStore()

			user1 := &account.User{Email: "User 1"}
			user2 := &account.User{Email: "User 2"}

			if err := store.AddUser(ctx, user1); err != nil {
				t.Fatal(err)
			}

			user2.ID = user1.ID

			if err := store.AddUser(ctx, user2); err != nil {
				t.Fatal(err)
			}

			if user2.ID == 0 || user2.ID == user1.ID {
				t.Error("want new id to be generated on add")
			}
		})

		t.Run("save and find", func(t *testing.T) {
			store := newStore()

			role1 := &account.Role{Name: "Foo"}
			if err := store.AddRole(ctx, role1); err != nil {
				t.Fatal(err)
			}

			tt := []struct {
				name string
				user account.User
				want error
			}{
				{"no data", account.User{}, nil},
				{"minimal data", account.User{Email: "Save user 1"}, nil},
				{"with recovery codes", account.User{Email: "Save user 2", RecoveryCodes: []string{"1", "2", "3", "4"}}, nil},
				{"with role", account.User{Email: "Save user roles", Roles: []*account.Role{role1}}, nil},
				{"with grants", account.User{Email: "Save user grants", Grants: []string{"abc"}}, nil},
				{"with denials", account.User{Email: "Save user denials", Denials: []string{"123"}}, nil},
				{"conflicting email", account.User{Email: "Save user 1"}, repo.ErrConflict},
				{"conflicting email with different casing", account.User{Email: "SAVE USER 1"}, repo.ErrConflict},
			}
			for i, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					user := account.User{Email: "New user " + strconv.Itoa(i)}
					if err := store.AddUser(ctx, &user); err != nil {
						t.Fatal(err)
					}

					tc.user.ID = user.ID

					err := store.SaveUser(ctx, &tc.user)
					if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
						t.Fatalf("want error: %v; got %v", tc.want, err)
					}

					if tc.want != nil {
						return
					}

					found, err := store.FindUserByID(ctx, tc.user.ID)
					if err != nil {
						t.Fatal(err)
					}

					accountUsersEqual(t, &tc.user, found)
				})
			}
		})

		t.Run("save roles", func(t *testing.T) {
			store := newStore()

			role1 := &account.Role{Name: "Foo"}
			if err := store.AddRole(ctx, role1); err != nil {
				t.Fatal(err)
			}

			role2 := &account.Role{Name: "Bar"}
			if err := store.AddRole(ctx, role2); err != nil {
				t.Fatal(err)
			}

			user := account.User{Email: "Email 1"}
			if err := store.AddUser(ctx, &user); err != nil {
				t.Fatal(err)
			}

			user.Roles = []*account.Role{role1, role2}
			if err := store.SaveUser(ctx, &user); err != nil {
				t.Fatal(err)
			}

			found, err := store.FindUserByID(ctx, user.ID)
			if err != nil {
				t.Fatal(err)
			}

			accountUsersEqual(t, &user, found)

			user.Roles = []*account.Role{role1}
			if err := store.SaveUser(ctx, &user); err != nil {
				t.Fatal(err)
			}

			found, err = store.FindUserByID(ctx, user.ID)
			if err != nil {
				t.Fatal(err)
			}

			accountUsersEqual(t, &user, found)
		})
	})

	t.Run("roles", func(t *testing.T) {
		t.Run("conflicts", func(t *testing.T) {
			store := newStore()

			role1 := &account.Role{Name: "Foo"}
			if err := store.AddRole(ctx, role1); err != nil {
				t.Fatal(err)
			}

			role2 := &account.Role{Name: "Bar"}
			if err := store.AddRole(ctx, role2); err != nil {
				t.Fatal(err)
			}
			role2.Name = role1.Name

			tt := []struct {
				name string
				err  error
			}{
				{"add", store.AddRole(ctx, role1)},
				{"save", store.SaveRole(ctx, role2)},
			}
			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					var conflicts *repo.ConflictError
					if !errors.As(tc.err, &conflicts) {
						t.Fatalf("want %T; got %T", conflicts, tc.err)
					}

					if conflicts.Get("name") == "" {
						t.Error("want name conflict error; got empty string")
					}
				})
			}
		})

		t.Run("add and find", func(t *testing.T) {
			store := newStore()

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
				{"conflicting name", account.Role{Name: "Name 1"}, repo.ErrConflict},
				{"conflicting name with different casing", account.Role{Name: "NAME 1"}, repo.ErrConflict},
			}
			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					err := store.AddRole(ctx, &tc.role)
					if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
						t.Fatalf("want error: %v; got %v", tc.want, err)
					}

					found, err := store.FindRoleByID(ctx, tc.role.ID)
					if err != nil {
						switch {
						case tc.want == nil:
							t.Fatal(err)

						case !errors.Is(err, repo.ErrNotFound):
							t.Errorf("want error: %v; got %v", repo.ErrNotFound, err)
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
			store := newStore()

			role := &account.Role{Name: account.GenerateRoleName().String()}
			if err := store.AddRole(ctx, role); err != nil {
				t.Fatal(err)
			}

			if err := store.RemoveRole(ctx, role.ID); err != nil {
				t.Fatalf("want error: <nil>; got %v", err)
			}

			if _, err := store.FindRoleByID(ctx, role.ID); err == nil {
				t.Errorf("want error: %v; got <nil>", repo.ErrNotFound)
			}
		})

		t.Run("generate new id on add", func(t *testing.T) {
			store := newStore()

			role1 := &account.Role{Name: "Role 1"}
			role2 := &account.Role{Name: "Role 2"}

			if err := store.AddRole(ctx, role1); err != nil {
				t.Fatal(err)
			}

			role2.ID = role1.ID

			if err := store.AddRole(ctx, role2); err != nil {
				t.Fatal(err)
			}

			if role2.ID == 0 || role2.ID == role1.ID {
				t.Error("want new id to be generated on add")
			}
		})

		t.Run("save and find", func(t *testing.T) {
			store := newStore()

			tt := []struct {
				name string
				role account.Role
				want error
			}{
				{"no data", account.Role{}, nil},
				{"minimal data", account.Role{Name: "Save name 1"}, nil},
				{"conflicting name", account.Role{Name: "Save name 1"}, repo.ErrConflict},
				{"conflicting name with different casing", account.Role{Name: "SAVE NAME 1"}, repo.ErrConflict},
				{"with permissions", account.Role{Name: "Save name 2", Permissions: []string{"1", "2", "3"}}, nil},
			}
			for i, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					role := account.Role{Name: "New role " + strconv.Itoa(i)}
					if err := store.AddRole(ctx, &role); err != nil {
						t.Fatal(err)
					}

					if tc.role.ID == 0 {
						tc.role.ID = role.ID
					}

					err := store.SaveRole(ctx, &tc.role)
					if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
						t.Fatalf("want error: %v; got %v", tc.want, err)
					}

					if tc.want != nil {
						return
					}

					found, err := store.FindRoleByID(ctx, tc.role.ID)
					if err != nil {
						t.Fatal(err)
					}

					accountRolesEqual(t, &tc.role, found)
				})
			}
		})
	})
}

func accountUsersEqual(t *testing.T, want, got *account.User) {
	t.Helper()

	if want, got := want.ID, got.ID; want != got {
		t.Errorf("want id to be %v; got %v", want, got)
	}
	if want, got := want.Email, got.Email; want != got {
		t.Errorf("want email to be %v; got %v", want, got)
	}
	if want, got := want.HashedPassword, got.HashedPassword; !bytes.Equal(want, got) {
		t.Errorf("want hashed password to be %v; got %v", want, got)
	}
	if want, got := want.TOTPMethod, got.TOTPMethod; want != got {
		t.Errorf("want totp method to be %v; got %v", want, got)
	}
	if want, got := want.TOTPTel, got.TOTPTel; want != got {
		t.Errorf("want totp tel to be %v; got %v", want, got)
	}
	if want, got := want.TOTPKey, got.TOTPKey; !bytes.Equal(want, got) {
		t.Errorf("want totp key to be %v; got %v", want, got)
	}
	if want, got := want.TOTPAlgorithm, got.TOTPAlgorithm; want != got {
		t.Errorf("want totp algorithm to be %v; got %v", want, got)
	}
	if want, got := want.TOTPDigits, got.TOTPDigits; want != got {
		t.Errorf("want totp digits to be %v; got %v", want, got)
	}
	if want, got := want.TOTPPeriod, got.TOTPPeriod; want != got {
		t.Errorf("want totp period to be %v; got %v", want, got)
	}
	if want, got := want.TOTPVerifiedAt, got.TOTPVerifiedAt; !want.Equal(got) {
		t.Errorf("want totp verified at to be %v; got %v", want, got)
	}
	if want, got := want.TOTPActivatedAt, got.TOTPActivatedAt; !want.Equal(got) {
		t.Errorf("want totp activated at to be %v; got %v", want, got)
	}
	if want, got := want.SignedUpAt, got.SignedUpAt; !want.Equal(got) {
		t.Errorf("want signed up at to be %v; got %v", want, got)
	}
	if want, got := want.ActivatedAt, got.ActivatedAt; !want.Equal(got) {
		t.Errorf("want activated at to be %v; got %v", want, got)
	}
	if want, got := want.LastSignedInAt, got.LastSignedInAt; !want.Equal(got) {
		t.Errorf("want last signed in at to be %v; got %v", want, got)
	}
	if want, got := want.LastSignedInMethod, got.LastSignedInMethod; want != got {
		t.Errorf("want last signed in method to be %v; got %v", want, got)
	}
	if want, got := want.Roles, got.Roles; len(want) != len(got) {
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
	if want, got := want.Grants, got.Grants; len(want) != len(got) {
		t.Errorf("want %v grants; got %v", len(want), len(got))
	} else {
		sort.Strings(want)
		sort.Strings(got)

		for i, wantGrant := range want {
			gotGrant := got[i]

			if wantGrant != gotGrant {
				t.Errorf("want grant %q; got %q", wantGrant, gotGrant)
			}
		}
	}
	if want, got := want.Denials, got.Denials; len(want) != len(got) {
		t.Errorf("want %v denials; got %v", len(want), len(got))
	} else {
		sort.Strings(want)
		sort.Strings(got)

		for i, wantDenial := range want {
			gotDenial := got[i]

			if wantDenial != gotDenial {
				t.Errorf("want denial %q; got %q", wantDenial, gotDenial)
			}
		}
	}
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
		sort.Strings(want)
		sort.Strings(got)

		for i, wantPermission := range want {
			gotPermission := got[i]

			if wantPermission != gotPermission {
				t.Errorf("want permission %q; got %q", wantPermission, gotPermission)
			}
		}
	}
}
