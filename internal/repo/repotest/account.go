package repotest

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/repo"
)

func Account(ctx context.Context, t *testing.T, store account.ReadWriter) {
	t.Run("add and find a new users", func(t *testing.T) {
		tt := []struct {
			name string
			user *account.User
			want error
		}{
			{"no data", &account.User{}, repo.ErrInvalidInput},
			{"minimal data whitespace", &account.User{Email: "    "}, repo.ErrInvalidInput},
			{"minimal data", &account.User{Email: "Email 1"}, nil},
			{"maximal data", &account.User{
				Email:           "Email 2",
				HashedPassword:  []byte("HashedPassword"),
				TOTPMethod:      "TOTPMethod",
				TOTPTelephone:   "TOTPTelephone",
				TOTPKey:         []byte("TOTPKey"),
				TOTPAlgorithm:   "TOTPAlgorithm",
				TOTPDigits:      123,
				TOTPPeriod:      456,
				TOTPVerifiedAt:  time.Now(),
				TOTPActivatedAt: time.Now(),
				SignedUpAt:      time.Now(),
				ActivatedAt:     time.Now(),
				LastSignedInAt:  time.Now(),
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

				if want, got := tc.user.ID, found.ID; want != got {
					t.Errorf("want id to be %v; got %v", want, got)
				}
				if want, got := tc.user.Email, found.Email; want != got {
					t.Errorf("want email to be %v; got %v", want, got)
				}
				if want, got := tc.user.HashedPassword, found.HashedPassword; !bytes.Equal(want, got) {
					t.Errorf("want hashed password to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPMethod, found.TOTPMethod; want != got {
					t.Errorf("want totp method to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPTelephone, found.TOTPTelephone; want != got {
					t.Errorf("want totp telephone to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPKey, found.TOTPKey; !bytes.Equal(want, got) {
					t.Errorf("want totp key to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPAlgorithm, found.TOTPAlgorithm; want != got {
					t.Errorf("want totp algorithm to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPDigits, found.TOTPDigits; want != got {
					t.Errorf("want totp digits to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPPeriod, found.TOTPPeriod; want != got {
					t.Errorf("want totp period to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPVerifiedAt, found.TOTPVerifiedAt; !want.Equal(got) {
					t.Errorf("want totp verified at to be %v; got %v", want, got)
				}
				if want, got := tc.user.TOTPActivatedAt, found.TOTPActivatedAt; !want.Equal(got) {
					t.Errorf("want totp activated at to be %v; got %v", want, got)
				}
				if want, got := tc.user.SignedUpAt, found.SignedUpAt; !want.Equal(got) {
					t.Errorf("want signed up at to be %v; got %v", want, got)
				}
				if want, got := tc.user.ActivatedAt, found.ActivatedAt; !want.Equal(got) {
					t.Errorf("want activated at to be %v; got %v", want, got)
				}
				if want, got := tc.user.LastSignedInAt, found.LastSignedInAt; !want.Equal(got) {
					t.Errorf("want last signed in at to be %v; got %v", want, got)
				}
			})
		}

		t.Run("generate new id on add", func(t *testing.T) {
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
	})

	t.Run("add and find a new roles", func(t *testing.T) {
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

				if want, got := tc.role.ID, found.ID; want != got {
					t.Errorf("want id to be %v; got %v", want, got)
				}
				if want, got := tc.role.Name, found.Name; want != got {
					t.Errorf("want name to be %v; got %v", want, got)
				}
			})
		}

		t.Run("generate new id on add", func(t *testing.T) {
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
	})
}
