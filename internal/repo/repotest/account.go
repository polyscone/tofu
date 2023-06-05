package repotest

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
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

			tt := []struct {
				name string
				user *account.User
				want error
			}{
				{"no data", &account.User{}, repo.ErrInvalidInput},
				{"minimal data whitespace", &account.User{Email: "    "}, repo.ErrInvalidInput},
				{"minimal data", &account.User{Email: "Email 1"}, nil},
				{"maximal data", &account.User{
					Email:              "Email 2",
					HashedPassword:     []byte("HashedPassword"),
					TOTPMethod:         "TOTPMethod",
					TOTPTelephone:      "TOTPTelephone",
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
				}, nil},
				{"conflicting email", &account.User{Email: "Email 1"}, repo.ErrConflict},
				{"conflicting email with different casing", &account.User{Email: "EMAIL 1"}, repo.ErrConflict},
			}
			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					err := store.AddUser(ctx, tc.user)
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

					accountUsersEqual(t, tc.user, found)
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

			tt := []struct {
				name string
				user *account.User
				want error
			}{
				{"no data", &account.User{}, repo.ErrInvalidInput},
				{"minimal data whitespace", &account.User{Email: "    "}, repo.ErrInvalidInput},
				{"minimal data", &account.User{Email: "Save user 1"}, nil},
				{"conflicting email", &account.User{Email: "Save user 1"}, repo.ErrConflict},
				{"conflicting email with different casing", &account.User{Email: "SAVE USER 1"}, repo.ErrConflict},
			}
			for i, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					user := &account.User{Email: "New user " + strconv.Itoa(i)}
					if err := store.AddUser(ctx, user); err != nil {
						t.Fatal(err)
					}

					if tc.user.ID == 0 {
						tc.user.ID = user.ID
					}

					err := store.SaveUser(ctx, tc.user)
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

					accountUsersEqual(t, tc.user, found)
				})
			}
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
				role *account.Role
				want error
			}{
				{"no data", &account.Role{}, repo.ErrInvalidInput},
				{"minimal data whitespace", &account.Role{Name: "    "}, repo.ErrInvalidInput},
				{"minimal data", &account.Role{Name: "Name 1"}, nil},
				{"conflicting name", &account.Role{Name: "Name 1"}, repo.ErrConflict},
				{"conflicting name with different casing", &account.Role{Name: "NAME 1"}, repo.ErrConflict},
			}
			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					err := store.AddRole(ctx, tc.role)
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

					accountRolesEqual(t, tc.role, found)
				})
			}
		})

		t.Run("add and remove", func(t *testing.T) {
			store := newStore()

			role := &account.Role{Name: text.GenerateName().String()}
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
				role *account.Role
				want error
			}{
				{"no data", &account.Role{}, repo.ErrInvalidInput},
				{"minimal data whitespace", &account.Role{Name: "    "}, repo.ErrInvalidInput},
				{"minimal data", &account.Role{Name: "Save name 1"}, nil},
				{"conflicting name", &account.Role{Name: "Save name 1"}, repo.ErrConflict},
				{"conflicting name with different casing", &account.Role{Name: "SAVE NAME 1"}, repo.ErrConflict},
			}
			for i, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					role := &account.Role{Name: "New role " + strconv.Itoa(i)}
					if err := store.AddRole(ctx, role); err != nil {
						t.Fatal(err)
					}

					if tc.role.ID == 0 {
						tc.role.ID = role.ID
					}

					err := store.SaveRole(ctx, tc.role)
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

					accountRolesEqual(t, tc.role, found)
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
	if want, got := want.TOTPTelephone, got.TOTPTelephone; want != got {
		t.Errorf("want totp telephone to be %v; got %v", want, got)
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
}

func accountRolesEqual(t *testing.T, want, got *account.Role) {
	t.Helper()

	if want, got := want.ID, got.ID; want != got {
		t.Errorf("want id to be %v; got %v", want, got)
	}
	if want, got := want.Name, got.Name; want != got {
		t.Errorf("want name to be %v; got %v", want, got)
	}
}
