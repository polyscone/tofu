package account

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app"
)

type Reader interface {
	FindRoleByID(ctx context.Context, id int) (*Role, error)
	FindRoleByName(ctx context.Context, name string) (*Role, error)

	CountUsers(ctx context.Context) (int, error)
	CountUsersByRoleID(ctx context.Context, roleID int) (int, error)
	FindUserByID(ctx context.Context, id int) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)

	FindSignInAttemptLogByEmail(ctx context.Context, email string) (*SignInAttemptLog, error)
}

type Writer interface {
	AddRole(ctx context.Context, role *Role) error
	SaveRole(ctx context.Context, role *Role) error
	RemoveRole(ctx context.Context, roleID int) error

	AddUser(ctx context.Context, user *User) error
	SaveUser(ctx context.Context, user *User) error

	SaveSignInAttemptLog(ctx context.Context, log *SignInAttemptLog) error
}

type ReadWriter interface {
	Reader
	Writer
}

func TestRepo(ctx context.Context, t *testing.T, newRepo func() ReadWriter) {
	t.Run("roles", func(t *testing.T) { testRepoRoles(ctx, t, newRepo) })
	t.Run("users", func(t *testing.T) { testRepoUsers(ctx, t, newRepo) })
	t.Run("sign in attempts", func(t *testing.T) { testSignInAttemptLogs(ctx, t, newRepo) })
}

func testRepoRoles(ctx context.Context, t *testing.T, newRepo func() ReadWriter) {
	t.Run("add and find", func(t *testing.T) {
		repo := newRepo()

		tt := []struct {
			name string
			role Role
			want error
		}{
			{"no data", Role{}, nil},
			{"minimal data", Role{Name: "Name 1"}, nil},
			{"maximal data", Role{
				Name:        "With description",
				Description: "This is a basic description.",
				Permissions: []string{"1", "2", "3"},
			}, nil},
			{"conflicting name", Role{Name: "Name 1"}, app.ErrConflict},
			{"conflicting name with different casing", Role{Name: "NAME 1"}, app.ErrConflict},
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

					case !errors.Is(err, app.ErrNotFound):
						t.Errorf("want error: %v; got %v", app.ErrNotFound, err)
					}
				}

				if tc.want != nil {
					return
				}

				testRolesEqual(t, &tc.role, found)
			})
		}
	})

	t.Run("add and remove", func(t *testing.T) {
		repo := newRepo()

		role := &Role{Name: "Role name"}
		if err := repo.AddRole(ctx, role); err != nil {
			t.Fatal(err)
		}

		if err := repo.RemoveRole(ctx, role.ID); err != nil {
			t.Fatalf("want error: <nil>; got %v", err)
		}

		if _, err := repo.FindRoleByID(ctx, role.ID); err == nil {
			t.Errorf("want error: %v; got <nil>", app.ErrNotFound)
		}
	})

	t.Run("generate new id on add", func(t *testing.T) {
		repo := newRepo()

		role1 := &Role{Name: "Role 1"}
		role2 := &Role{Name: "Role 2"}

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
			role Role
			want error
		}{
			{"no data", Role{}, nil},
			{"minimal data", Role{Name: "Save name 1"}, nil},
			{"conflicting name", Role{Name: "Save name 1"}, app.ErrConflict},
			{"conflicting name with different casing", Role{Name: "SAVE NAME 1"}, app.ErrConflict},
			{"with permissions", Role{Name: "Save name 2", Permissions: []string{"1", "2", "3"}}, nil},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				role := Role{Name: "New role " + strconv.Itoa(i)}
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

				testRolesEqual(t, &tc.role, found)
			})
		}
	})

	t.Run("conflicts", func(t *testing.T) {
		repo := newRepo()

		role1 := &Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2 := &Role{Name: "Bar"}
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
				var conflict *app.ConflictError
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

func testRolesEqual(t *testing.T, want, got *Role) {
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

func testRepoUsers(ctx context.Context, t *testing.T, newRepo func() ReadWriter) {
	t.Run("add and find", func(t *testing.T) {
		repo := newRepo()

		role1 := &Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2 := &Role{Name: "Bar"}
		if err := repo.AddRole(ctx, role2); err != nil {
			t.Fatal(err)
		}

		tt := []struct {
			name string
			user User
			want error
		}{
			{"no data", User{}, nil},
			{"minimal data", User{Email: "Email 1"}, nil},
			{"maximal data", User{
				Email:                   "Email 2",
				HashedPassword:          []byte("HashedPassword"),
				TOTPMethod:              "TOTPMethod",
				TOTPTel:                 "TOTPTel",
				TOTPKey:                 []byte("TOTPKey"),
				TOTPAlgorithm:           "TOTPAlgorithm",
				TOTPDigits:              123,
				TOTPPeriod:              456,
				TOTPVerifiedAt:          time.Now().Add(-1 * time.Second),
				TOTPActivatedAt:         time.Now().Add(-2 * time.Second),
				TOTPResetRequestedAt:    time.Now().Add(-3 * time.Second),
				TOTPResetApprovedAt:     time.Now().Add(-4 * time.Second),
				InvitedAt:               time.Now().Add(-5 * time.Second),
				SignedUpAt:              time.Now().Add(-6 * time.Second),
				SignedUpSystem:          "site",
				SignedUpMethod:          "web form",
				VerifiedAt:              time.Now().Add(-7 * time.Second),
				ActivatedAt:             time.Now().Add(-8 * time.Second),
				LastSignInAttemptAt:     time.Now().Add(-9 * time.Second),
				LastSignInAttemptSystem: "site",
				LastSignInAttemptMethod: "google",
				LastSignedInAt:          time.Now().Add(-10 * time.Second),
				LastSignedInSystem:      "pwa",
				LastSignedInMethod:      "web form",
				SuspendedAt:             time.Now().Add(-11 * time.Second),
				SuspendedReason:         "They should not be able to access the system anymore",
				HashedRecoveryCodes:     [][]byte{[]byte("1"), []byte("2"), []byte("3")},
				Roles:                   []*Role{role1, role2},
				Grants:                  []string{"a", "b"},
				Denials:                 []string{"b", "c", "d"},
			}, nil},
			{"conflicting email", User{Email: "Email 1"}, app.ErrConflict},
			{"conflicting email with different casing", User{Email: "EMAIL 1"}, app.ErrConflict},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := repo.AddUser(ctx, &tc.user)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				found, err := repo.FindUserByID(ctx, tc.user.ID)
				if err != nil {
					switch {
					case tc.want == nil:
						t.Fatal(err)

					case !errors.Is(err, app.ErrNotFound):
						t.Errorf("want error: %v; got %v", app.ErrNotFound, err)
					}
				}

				if tc.want != nil {
					return
				}

				testUsersEqual(t, &tc.user, found)
			})
		}
	})

	t.Run("generate new id on add", func(t *testing.T) {
		repo := newRepo()

		user1 := &User{Email: "User 1"}
		user2 := &User{Email: "User 2"}

		if err := repo.AddUser(ctx, user1); err != nil {
			t.Fatal(err)
		}

		user2.ID = user1.ID

		if err := repo.AddUser(ctx, user2); err != nil {
			t.Fatal(err)
		}

		if user2.ID == 0 || user2.ID == user1.ID {
			t.Error("want new id to be generated on add")
		}
	})

	t.Run("save and find", func(t *testing.T) {
		repo := newRepo()

		role1 := &Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		tt := []struct {
			name string
			user User
			want error
		}{
			{"no data", User{}, nil},
			{"minimal data", User{Email: "Save user 1"}, nil},
			{"with suspension", User{Email: "Suspended user 1", SuspendedAt: time.Now().Add(-1 * time.Second), SuspendedReason: "Foo bar"}, nil},
			{"with recovery codes", User{Email: "Save user 2", HashedRecoveryCodes: [][]byte{[]byte("1"), []byte("2"), []byte("3")}}, nil},
			{"with role", User{Email: "Save user roles", Roles: []*Role{role1}}, nil},
			{"with grants", User{Email: "Save user grants", Grants: []string{"abc"}}, nil},
			{"with denials", User{Email: "Save user denials", Denials: []string{"123"}}, nil},
			{"conflicting email", User{Email: "Save user 1"}, app.ErrConflict},
			{"conflicting email with different casing", User{Email: "SAVE USER 1"}, app.ErrConflict},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := User{Email: "New user " + strconv.Itoa(i)}
				if err := repo.AddUser(ctx, &user); err != nil {
					t.Fatal(err)
				}

				tc.user.ID = user.ID

				err := repo.SaveUser(ctx, &tc.user)
				if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("want error: %v; got %v", tc.want, err)
				}

				if tc.want != nil {
					return
				}

				found, err := repo.FindUserByID(ctx, tc.user.ID)
				if err != nil {
					t.Fatal(err)
				}

				testUsersEqual(t, &tc.user, found)
			})
		}
	})

	t.Run("save with roles", func(t *testing.T) {
		repo := newRepo()

		role1 := &Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2 := &Role{Name: "Bar"}
		if err := repo.AddRole(ctx, role2); err != nil {
			t.Fatal(err)
		}

		user := User{Email: "Email 1"}
		if err := repo.AddUser(ctx, &user); err != nil {
			t.Fatal(err)
		}

		user.Roles = []*Role{role1, role2}
		if err := repo.SaveUser(ctx, &user); err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		testUsersEqual(t, &user, found)

		user.Roles = []*Role{role1}
		if err := repo.SaveUser(ctx, &user); err != nil {
			t.Fatal(err)
		}

		found, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		testUsersEqual(t, &user, found)
	})

	t.Run("conflicts", func(t *testing.T) {
		repo := newRepo()

		user1 := &User{Email: "Foo"}
		if err := repo.AddUser(ctx, user1); err != nil {
			t.Fatal(err)
		}

		user2 := &User{Email: "Bar"}
		if err := repo.AddUser(ctx, user2); err != nil {
			t.Fatal(err)
		}
		user2.Email = user1.Email

		tt := []struct {
			name string
			err  error
		}{
			{"add", repo.AddUser(ctx, user1)},
			{"save", repo.SaveUser(ctx, user2)},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				var conflict *app.ConflictError
				if !errors.As(tc.err, &conflict) {
					t.Fatalf("want %T; got %T", conflict, tc.err)
				}

				if conflict.Get("email") == "" {
					t.Error("want email conflict error; got empty string")
				}
			})
		}
	})
}

func testUsersEqual(t *testing.T, want, got *User) {
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
	if want, got := want.TOTPResetRequestedAt, got.TOTPResetRequestedAt; !want.Equal(got) {
		t.Errorf("want totp reset requested at to be %v; got %v", want, got)
	}
	if want, got := want.TOTPResetApprovedAt, got.TOTPResetApprovedAt; !want.Equal(got) {
		t.Errorf("want totp reset approved at to be %v; got %v", want, got)
	}
	if want, got := want.InvitedAt, got.InvitedAt; !want.Equal(got) {
		t.Errorf("want invited at to be %v; got %v", want, got)
	}
	if want, got := want.SignedUpAt, got.SignedUpAt; !want.Equal(got) {
		t.Errorf("want signed up at to be %v; got %v", want, got)
	}
	if want, got := want.SignedUpSystem, got.SignedUpSystem; want != got {
		t.Errorf("want signed up app to be %v; got %v", want, got)
	}
	if want, got := want.SignedUpMethod, got.SignedUpMethod; want != got {
		t.Errorf("want signed up method to be %v; got %v", want, got)
	}
	if want, got := want.VerifiedAt, got.VerifiedAt; !want.Equal(got) {
		t.Errorf("want verified at to be %v; got %v", want, got)
	}
	if want, got := want.ActivatedAt, got.ActivatedAt; !want.Equal(got) {
		t.Errorf("want activated at to be %v; got %v", want, got)
	}
	if want, got := want.LastSignInAttemptAt, got.LastSignInAttemptAt; !want.Equal(got) {
		t.Errorf("want last sign in attempt at to be %v; got %v", want, got)
	}
	if want, got := want.LastSignInAttemptSystem, got.LastSignInAttemptSystem; want != got {
		t.Errorf("want last sign in attempt system to be %v; got %v", want, got)
	}
	if want, got := want.LastSignInAttemptMethod, got.LastSignInAttemptMethod; want != got {
		t.Errorf("want last sign in attempt method to be %v; got %v", want, got)
	}
	if want, got := want.LastSignedInAt, got.LastSignedInAt; !want.Equal(got) {
		t.Errorf("want last signed in at to be %v; got %v", want, got)
	}
	if want, got := want.LastSignedInSystem, got.LastSignedInSystem; want != got {
		t.Errorf("want last signed in system to be %v; got %v", want, got)
	}
	if want, got := want.LastSignedInMethod, got.LastSignedInMethod; want != got {
		t.Errorf("want last signed in method to be %v; got %v", want, got)
	}
	if want, got := want.SuspendedAt, got.SuspendedAt; !want.Equal(got) {
		t.Errorf("want suspended at to be %v; got %v", want, got)
	}
	if want, got := want.SuspendedReason, got.SuspendedReason; want != got {
		t.Errorf("want suspended reason to be %v; got %v", want, got)
	}
	if want, got := want.Roles, got.Roles; len(want) != len(got) {
		t.Errorf("want %v roles; got %v", len(want), len(got))
	} else {
		slices.SortFunc(want, func(a, b *Role) int { return cmp.Compare(a.ID, b.ID) })
		slices.SortFunc(got, func(a, b *Role) int { return cmp.Compare(a.ID, b.ID) })

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
		slices.Sort(want)
		slices.Sort(got)

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
		slices.Sort(want)
		slices.Sort(got)

		for i, wantDenial := range want {
			gotDenial := got[i]

			if wantDenial != gotDenial {
				t.Errorf("want denial %q; got %q", wantDenial, gotDenial)
			}
		}
	}
}

func testSignInAttemptLogs(ctx context.Context, t *testing.T, newRepo func() ReadWriter) {
	t.Run("find and save", func(t *testing.T) {
		repo := newRepo()

		log, err := repo.FindSignInAttemptLogByEmail(ctx, "foo@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "foo@example.com", log.Email; want != got {
			t.Errorf("want email to be %q; got %q", want, got)
		}
		if want, got := 0, log.Attempts; want != got {
			t.Errorf("want attempts to be %v; got %v", want, got)
		}
		if !log.LastAttemptAt.IsZero() {
			t.Errorf("want last attempt at to be zero; got %v", log.LastAttemptAt)
		}

		log.Attempts = 5
		log.LastAttemptAt = time.Now()
		if err := repo.SaveSignInAttemptLog(ctx, log); err != nil {
			t.Fatal(err)
		}

		log, err = repo.FindSignInAttemptLogByEmail(ctx, "foo@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "foo@example.com", log.Email; want != got {
			t.Errorf("want email to be %q; got %q", want, got)
		}
		if want, got := 5, log.Attempts; want != got {
			t.Errorf("want attempts to be %v; got %v", want, got)
		}
		if log.LastAttemptAt.IsZero() {
			t.Error("want last attempt at to be populated; got zero")
		}
	})
}
