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
)

type regenerateRecoveryCodesGuard struct {
	value bool
}

func (g regenerateRecoveryCodesGuard) CanRegenerateRecoveryCodes(userID int) bool {
	return g.value
}

func TestRegenRecoveryCodes(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	activatedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, ActivateTOTP: true})

	validGuard := regenerateRecoveryCodesGuard{value: true}
	invalidGuard := regenerateRecoveryCodesGuard{value: false}

	t.Run("success with user with verified TOTP", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		originals := activatedTOTP.RecoveryCodes

		alg := errors.Must(otp.NewAlgorithm(activatedTOTP.TOTPAlgorithm))
		tb := errors.Must(otp.NewTimeBased(activatedTOTP.TOTPDigits, alg, time.Unix(0, 0), activatedTOTP.TOTPPeriod))
		totp := errors.Must(tb.Generate(activatedTOTP.TOTPKey, time.Now()))

		err := svc.RegenerateRecoveryCodes(ctx, validGuard, activatedTOTP.ID, totp)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.RecoveryCodesRegenerated{Email: activatedTOTP.Email})

		user := errors.Must(store.FindUserByID(ctx, activatedTOTP.ID))

		if len(user.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.RecoveryCodes {
				if len(rc) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}

		if want, got := len(originals), len(user.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, original := range originals {
				for _, rc := range user.RecoveryCodes {
					if original == rc {
						t.Errorf("want different codes; got %q and %q", original, rc)
					}
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    regenerateRecoveryCodesGuard
			userID   int
			totpUser *account.User
			want     error
		}{
			{"unauthorised", invalidGuard, 0, activatedTOTP, app.ErrUnauthorised},
			{"TOTP not setup or verified", validGuard, activated.ID, nil, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					alg := errors.Must(otp.NewAlgorithm(tc.totpUser.TOTPAlgorithm))
					tb := errors.Must(otp.NewTimeBased(tc.totpUser.TOTPDigits, alg, time.Unix(0, 0), tc.totpUser.TOTPPeriod))
					totp = errors.Must(tb.Generate(tc.totpUser.TOTPKey, time.Now()))
				}

				err := svc.RegenerateRecoveryCodes(ctx, tc.guard, tc.userID, totp)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
