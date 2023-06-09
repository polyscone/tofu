package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/repo"
)

func TestSignInWithTOTP(t *testing.T) {
	t.Run("success for valid user id and correct totp", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "bob@jones.com", ActivateTOTP: true})

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		err := svc.SignInWithPassword(ctx, user.Email, "password")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		user, err = store.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		alg := errors.Must(otp.NewAlgorithm(user.TOTPAlgorithm))
		tb := errors.Must(otp.NewTimeBased(user.TOTPDigits, alg, time.Unix(0, 0), user.TOTPPeriod))
		totp := errors.Must(tb.Generate(user.TOTPKey, time.Now()))

		otp.CleanUsedTOTP(totp)

		err = svc.SignInWithTOTP(ctx, user.ID, totp)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		events.Expect(account.SignedInWithTOTP{Email: user.Email})

		user, err = store.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Activate: true})
		user2 := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", SetupTOTP: true})
		user3 := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", VerifyTOTP: true})
		user4 := MustAddUser(t, ctx, store, TestUser{Email: "bob@jones.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			userID   int
			totpUser *account.User
			want     error
		}{
			{"empty user id correct TOTP", 0, user4, repo.ErrNotFound},
			{"empty user id incorrect TOTP", 0, nil, repo.ErrNotFound},
			{"activated user id incorrect TOTP", user4.ID, nil, app.ErrInvalidInput},
			{"activated user id unverified correct TOTP", user2.ID, user2, app.ErrBadRequest},
			{"activated user id without TOTP setup", user1.ID, nil, app.ErrBadRequest},
			{"activated user id without TOTP activated", user3.ID, user3, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					alg := errors.Must(otp.NewAlgorithm(tc.totpUser.TOTPAlgorithm))
					tb := errors.Must(otp.NewTimeBased(tc.totpUser.TOTPDigits, alg, time.Unix(0, 0), tc.totpUser.TOTPPeriod))
					totp = errors.Must(tb.Generate(tc.totpUser.TOTPKey, time.Now()))
				}

				err := svc.SignInWithTOTP(ctx, tc.userID, totp)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})

	t.Run("properties", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "bob@jones.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(totp account.TOTP) error {
			err := svc.SignInWithTOTP(ctx, user.ID, totp.String())
			if err == nil {
				events.Expect(account.SignedInWithTOTP{Email: user.Email})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(totp account.TOTP) bool {
				err := execute(totp)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid totp input", func(t *testing.T) {
			quick.Check(t, func(totp quick.Invalid[account.TOTP]) bool {
				err := execute(totp.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
