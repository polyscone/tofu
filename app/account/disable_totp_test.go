package account_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/testutil"
)

type disableTOTPGuard struct {
	value bool
}

func (g disableTOTPGuard) CanDisableTOTP(userID int) bool {
	return g.value
}

func TestDisableTOTP(t *testing.T) {
	validGuard := disableTOTPGuard{value: true}
	invalidGuard := disableTOTPGuard{value: false}

	t.Run("success with correct password", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", SetupTOTPTel: true, ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.DisableTOTP(ctx, validGuard, user.ID, "password")
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPDisabled{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))
		if want, got := []byte(nil), user.TOTPKey; !bytes.Equal(want, got) {
			t.Errorf("want TOTP key %q; got %q", want, got)
		}
		if user.TOTPMethod != account.TOTPMethodNone.String() {
			t.Error("want TOTP method to be cleared")
		}
		if user.TOTPTel != "" {
			t.Error("want TOTP tel to be cleared")
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
		if want, got := 0, len(user.HashedRecoveryCodes); want != got {
			t.Error("want recovery codes to be cleared")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", SetupTOTPTel: true, VerifyTOTP: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", SetupTOTPTel: true, ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    disableTOTPGuard
			userID   int
			password string
			want     error
		}{
			{"invalid guard", invalidGuard, 0, "", app.ErrForbidden},
			{"incorrect password", validGuard, user2.ID, "12345678", nil},
			{"unactivated TOTP", validGuard, user1.ID, "password", nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.DisableTOTP(ctx, tc.guard, tc.userID, tc.password)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
