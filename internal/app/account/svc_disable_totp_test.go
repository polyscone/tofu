package account_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type disableTOTPGuard struct {
	value bool
}

func (g disableTOTPGuard) CanDisableTOTP(userID int) bool {
	return g.value
}

func TestDisableTOTP(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, SetupTOTPTelephone: true, VerifyTOTP: true})
	activatedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", Password: password, SetupTOTPTelephone: true, ActivateTOTP: true})

	validGuard := disableTOTPGuard{value: true}
	invalidGuard := disableTOTPGuard{value: false}

	t.Run("success with correct password", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.AuthenticatedWithPassword{Email: activatedTOTP.Email})
		events.Expect(account.DisabledTOTP{Email: activatedTOTP.Email})

		err := svc.DisableTOTP(ctx, validGuard, activatedTOTP.ID, password)
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, activatedTOTP.ID))
		if want, got := []byte(nil), user.TOTPKey; !bytes.Equal(want, got) {
			t.Errorf("want %q; got %q", want, got)
		}
		if user.TOTPMethod != account.TOTPMethodNone.String() {
			t.Error("want TOTP method to be cleared")
		}
		if user.TOTPTelephone != "" {
			t.Error("want TOTP telephone to be cleared")
		}
		if user.TOTPKey != nil {
			t.Error("want TOTP key to be cleared")
		}
		if user.TOTPAlgorithm != "" {
			t.Error("want TOTP algorithm to be cleared")
		}
		if user.TOTPDigits != 0 {
			t.Error("want TOTP digits to be cleared")
		}
		if user.TOTPPeriod != 0 {
			t.Error("want TOTP period to be cleared")
		}
		if got := user.TOTPVerifiedAt; !got.IsZero() {
			t.Errorf("want TOTP verified at to be zero; got %v", got)
		}
		if got := user.TOTPActivatedAt; !got.IsZero() {
			t.Errorf("want TOTP activated at to be zero; got %v", got)
		}
		if want, got := 0, len(user.RecoveryCodes); want != got {
			t.Error("want recovery codes to be cleared")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    account.DisableTOTPGuard
			userID   int
			password string
			want     error
		}{
			{"unauthorised", invalidGuard, 0, "", app.ErrUnauthorised},
			{"incorrect password", validGuard, activatedTOTP.ID, "12345678", app.ErrBadRequest},
			{"unactivated TOTP", validGuard, verifiedTOTP.ID, password, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.DisableTOTP(ctx, tc.guard, tc.userID, tc.password)
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
