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

func TestAuthenticateWithTOTP(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	unverifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", Password: password, SetupTOTP: true})
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, VerifyTOTP: true})
	activatedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "bob@jones.com", Password: password, ActivateTOTP: true})

	t.Run("success for valid user id and correct totp", func(t *testing.T) {
		t.Run("last signed in should not update on password auth", func(t *testing.T) {
			if !activatedTOTP.LastSignedInAt.IsZero() {
				t.Errorf("want last signed in at to be zero; got %v", activatedTOTP.LastSignedInAt)
			}

			err := svc.AuthenticateWithPassword(ctx, activatedTOTP.Email, "password")
			if err != nil {
				t.Errorf("want <nil>; got %q", err)
			}

			activatedTOTP, err = store.FindUserByID(ctx, activatedTOTP.ID)
			if err != nil {
				t.Fatal(err)
			}

			if !activatedTOTP.LastSignedInAt.IsZero() {
				t.Errorf("want last signed in at to be zero; got %v", activatedTOTP.LastSignedInAt)
			}
		})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.AuthenticatedWithTOTP{Email: activatedTOTP.Email})

		alg := errors.Must(otp.NewAlgorithm(activatedTOTP.TOTPAlgorithm))
		tb := errors.Must(otp.NewTimeBased(activatedTOTP.TOTPDigits, alg, time.Unix(0, 0), activatedTOTP.TOTPPeriod))
		totp := errors.Must(tb.Generate(activatedTOTP.TOTPKey, time.Now()))

		otp.CleanUsedTOTP(totp)

		err := svc.AuthenticateWithTOTP(ctx, activatedTOTP.ID, totp)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		activatedTOTP, err = store.FindUserByID(ctx, activatedTOTP.ID)
		if err != nil {
			t.Fatal(err)
		}

		if activatedTOTP.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			userID   int
			totpUser *account.User
			want     error
		}{
			{"empty user id correct TOTP", 0, activatedTOTP, repo.ErrNotFound},
			{"empty user id incorrect TOTP", 0, nil, repo.ErrNotFound},
			{"activated user id incorrect TOTP", activatedTOTP.ID, nil, app.ErrInvalidInput},
			{"activated user id unverified correct TOTP", unverifiedTOTP.ID, unverifiedTOTP, app.ErrBadRequest},
			{"activated user id without TOTP setup", activated.ID, nil, app.ErrBadRequest},
			{"activated user id without TOTP activated", verifiedTOTP.ID, verifiedTOTP, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					alg := errors.Must(otp.NewAlgorithm(tc.totpUser.TOTPAlgorithm))
					tb := errors.Must(otp.NewTimeBased(tc.totpUser.TOTPDigits, alg, time.Unix(0, 0), tc.totpUser.TOTPPeriod))
					totp = errors.Must(tb.Generate(tc.totpUser.TOTPKey, time.Now()))
				}

				err := svc.AuthenticateWithTOTP(ctx, tc.userID, totp)
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
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(totp account.TOTP) error {
			err := svc.AuthenticateWithTOTP(ctx, activatedTOTP.ID, totp.String())
			if err == nil {
				events.Expect(account.AuthenticatedWithTOTP{Email: activatedTOTP.Email})
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
