package repotest

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/repository"
)

func AccountUsers(ctx context.Context, t *testing.T, newRepo func() account.ReadWriter) {
	t.Run("add and find", func(t *testing.T) {
		repo := newRepo()

		role1 := &account.Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2 := &account.Role{Name: "Bar"}
		if err := repo.AddRole(ctx, role2); err != nil {
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
				Email:                "Email 2",
				HashedPassword:       []byte("HashedPassword"),
				TOTPMethod:           "TOTPMethod",
				TOTPTel:              "TOTPTel",
				TOTPKey:              []byte("TOTPKey"),
				TOTPAlgorithm:        "TOTPAlgorithm",
				TOTPDigits:           123,
				TOTPPeriod:           456,
				TOTPVerifiedAt:       time.Now().Add(-1 * time.Second),
				TOTPActivatedAt:      time.Now().Add(-2 * time.Second),
				TOTPResetRequestedAt: time.Now().Add(-3 * time.Second),
				TOTPResetApprovedAt:  time.Now().Add(-4 * time.Second),
				InvitedAt:            time.Now().Add(-5 * time.Second),
				SignedUpAt:           time.Now().Add(-6 * time.Second),
				SignedUpSystem:       "site",
				SignedUpMethod:       "form",
				VerifiedAt:           time.Now().Add(-7 * time.Second),
				ActivatedAt:          time.Now().Add(-8 * time.Second),
				LastSignedInAt:       time.Now().Add(-9 * time.Second),
				LastSignedInMethod:   "form",
				SuspendedAt:          time.Now().Add(-9 * time.Second),
				SuspendedReason:      "They should not be able to access the system anymore",
				HashedRecoveryCodes:  [][]byte{[]byte("1"), []byte("2"), []byte("3")},
				Roles:                []*account.Role{role1, role2},
				Grants:               []string{"a", "b"},
				Denials:              []string{"b", "c", "d"},
			}, nil},
			{"conflicting email", account.User{Email: "Email 1"}, repository.ErrConflict},
			{"conflicting email with different casing", account.User{Email: "EMAIL 1"}, repository.ErrConflict},
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

					case !errors.Is(err, repository.ErrNotFound):
						t.Errorf("want error: %v; got %v", repository.ErrNotFound, err)
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
		repo := newRepo()

		user1 := &account.User{Email: "User 1"}
		user2 := &account.User{Email: "User 2"}

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

		role1 := &account.Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		tt := []struct {
			name string
			user account.User
			want error
		}{
			{"no data", account.User{}, nil},
			{"minimal data", account.User{Email: "Save user 1"}, nil},
			{"with suspension", account.User{Email: "Suspended user 1", SuspendedAt: time.Now().Add(-1 * time.Second), SuspendedReason: "Foo bar"}, nil},
			{"with recovery codes", account.User{Email: "Save user 2", HashedRecoveryCodes: [][]byte{[]byte("1"), []byte("2"), []byte("3")}}, nil},
			{"with role", account.User{Email: "Save user roles", Roles: []*account.Role{role1}}, nil},
			{"with grants", account.User{Email: "Save user grants", Grants: []string{"abc"}}, nil},
			{"with denials", account.User{Email: "Save user denials", Denials: []string{"123"}}, nil},
			{"conflicting email", account.User{Email: "Save user 1"}, repository.ErrConflict},
			{"conflicting email with different casing", account.User{Email: "SAVE USER 1"}, repository.ErrConflict},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := account.User{Email: "New user " + strconv.Itoa(i)}
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

				accountUsersEqual(t, &tc.user, found)
			})
		}
	})

	t.Run("save with roles", func(t *testing.T) {
		repo := newRepo()

		role1 := &account.Role{Name: "Foo"}
		if err := repo.AddRole(ctx, role1); err != nil {
			t.Fatal(err)
		}

		role2 := &account.Role{Name: "Bar"}
		if err := repo.AddRole(ctx, role2); err != nil {
			t.Fatal(err)
		}

		user := account.User{Email: "Email 1"}
		if err := repo.AddUser(ctx, &user); err != nil {
			t.Fatal(err)
		}

		user.Roles = []*account.Role{role1, role2}
		if err := repo.SaveUser(ctx, &user); err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		accountUsersEqual(t, &user, found)

		user.Roles = []*account.Role{role1}
		if err := repo.SaveUser(ctx, &user); err != nil {
			t.Fatal(err)
		}

		found, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		accountUsersEqual(t, &user, found)
	})

	t.Run("conflicts", func(t *testing.T) {
		repo := newRepo()

		user1 := &account.User{Email: "Foo"}
		if err := repo.AddUser(ctx, user1); err != nil {
			t.Fatal(err)
		}

		user2 := &account.User{Email: "Bar"}
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
				var conflict *repository.ConflictError
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
	if want, got := want.LastSignedInAt, got.LastSignedInAt; !want.Equal(got) {
		t.Errorf("want last signed in at to be %v; got %v", want, got)
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
		slices.SortFunc(want, func(a, b *account.Role) int { return cmp.Compare(a.ID, b.ID) })
		slices.SortFunc(got, func(a, b *account.Role) int { return cmp.Compare(a.ID, b.ID) })

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
