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

type verifyTOTPGuard struct {
	value bool
}

func (g verifyTOTPGuard) CanVerifyTOTP(userID int) bool {
	return g.value
}

func TestVerifyTOTP(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, Activate: true})
	unverifiedAppTOTPUser := MustAddUser(t, ctx, store, TestUser{Email: "jane@doe.com", Password: password, SetupTOTP: true})
	unverifiedSMSTOTPUser := MustAddUser(t, ctx, store, TestUser{Email: "baz@qux.com", Password: password, SetupTOTPTelephone: true})
	activatedAppTOTPUser := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", Password: password, ActivateTOTP: true})

	validGuard := verifyTOTPGuard{value: true}
	invalidGuard := verifyTOTPGuard{value: false}

	t.Run("success with activated user with app TOTP setup", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		alg := errors.Must(otp.NewAlgorithm(unverifiedAppTOTPUser.TOTPAlgorithm))
		tb := errors.Must(otp.NewTimeBased(unverifiedAppTOTPUser.TOTPDigits, alg, time.Unix(0, 0), unverifiedAppTOTPUser.TOTPPeriod))
		totp := errors.Must(tb.Generate(unverifiedAppTOTPUser.TOTPKey, time.Now()))

		// Deliberately set the method to SMS so we can check it's changed by the service
		unverifiedAppTOTPUser.TOTPMethod = account.TOTPMethodSMS.String()

		err := svc.VerifyTOTP(ctx, validGuard, unverifiedAppTOTPUser.ID, totp, account.TOTPMethodApp.String())
		if err != nil {
			t.Fatal(err)
		}

		unverifiedAppTOTPUser = errors.Must(store.FindUserByID(ctx, unverifiedAppTOTPUser.ID))

		if want, got := account.TOTPMethodApp.String(), unverifiedAppTOTPUser.TOTPMethod; want != got {
			t.Errorf("want TOTP method to be %q; got %q", want, got)
		}

		if len(unverifiedAppTOTPUser.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range unverifiedAppTOTPUser.RecoveryCodes {
				if len(rc.Code) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}
	})

	t.Run("success with activated user with SMS TOTP setup", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		alg := errors.Must(otp.NewAlgorithm(unverifiedSMSTOTPUser.TOTPAlgorithm))
		tb := errors.Must(otp.NewTimeBased(unverifiedSMSTOTPUser.TOTPDigits, alg, time.Unix(0, 0), unverifiedSMSTOTPUser.TOTPPeriod))
		totp := errors.Must(tb.Generate(unverifiedSMSTOTPUser.TOTPKey, time.Now()))

		// Deliberately set the method to app so we can check it's changed by the service
		unverifiedSMSTOTPUser.TOTPMethod = account.TOTPMethodSMS.String()

		err := svc.VerifyTOTP(ctx, validGuard, unverifiedSMSTOTPUser.ID, totp, account.TOTPMethodApp.String())
		if err != nil {
			t.Fatal(err)
		}

		unverifiedSMSTOTPUser = errors.Must(store.FindUserByID(ctx, unverifiedSMSTOTPUser.ID))

		if want, got := account.TOTPMethodApp.String(), unverifiedSMSTOTPUser.TOTPMethod; want != got {
			t.Errorf("want TOTP method to be %q; got %q", want, got)
		}

		if len(unverifiedSMSTOTPUser.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range unverifiedSMSTOTPUser.RecoveryCodes {
				if len(rc.Code) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    verifyTOTPGuard
			userID   int
			totpUser *account.User
			want     error
		}{
			{"unauthorised", invalidGuard, 0, nil, app.ErrUnauthorised},
			{"empty user id correct TOTP", validGuard, 0, unverifiedAppTOTPUser, repo.ErrNotFound},
			{"empty user id incorrect TOTP", validGuard, 0, nil, repo.ErrNotFound},
			{"no TOTP user id correct TOTP", validGuard, activated.ID, unverifiedAppTOTPUser, nil},
			{"already activated TOTP", validGuard, activatedAppTOTPUser.ID, activatedAppTOTPUser, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					alg := errors.Must(otp.NewAlgorithm(tc.totpUser.TOTPAlgorithm))
					tb := errors.Must(otp.NewTimeBased(tc.totpUser.TOTPDigits, alg, time.Unix(0, 0), tc.totpUser.TOTPPeriod))
					totp = errors.Must(tb.Generate(tc.totpUser.TOTPKey, time.Now()))
				}

				err := svc.VerifyTOTP(ctx, tc.guard, tc.userID, totp, account.TOTPMethodApp.String())
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
			return svc.VerifyTOTP(ctx, validGuard, activated.ID, totp.String(), account.TOTPMethodApp.String())
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
