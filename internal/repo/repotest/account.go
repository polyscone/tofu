package repotest

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
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
				TOTPVerifiedAt:  time.Now().UTC(),
				TOTPActivatedAt: time.Now().UTC(),
				SignedUpAt:      time.Now().UTC(),
				ActivatedAt:     time.Now().UTC(),
				LastSignedInAt:  time.Now().UTC(),
			}, nil},
			{"conflicting email", &account.User{Email: "Email 1"}, repo.ErrConflict},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := store.AddUser(ctx, tc.user)
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
	})

	t.Run("generate new id on add", func(t *testing.T) {
		email1 := text.GenerateEmail()
		user1 := account.NewUser(email1)

		email2 := text.GenerateEmail()
		user2 := account.NewUser(email2)

		password := account.GeneratePassword()
		hasher := testutil.NewPasswordHasher()
		if err := user1.SignUpWithPassword(password, hasher); err != nil {
			t.Fatal(err)
		}
		if err := user2.SignUpWithPassword(password, hasher); err != nil {
			t.Fatal(err)
		}

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
}
