package account_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/pkg/errsx"
	"github.com/polyscone/tofu/pkg/testutil"
)

type resetTOTPGuard struct {
	value bool
}

func (g resetTOTPGuard) CanResetTOTP(userID string) bool {
	return g.value
}

func TestResetTOTP(t *testing.T) {
	validGuard := resetTOTPGuard{value: true}
	invalidGuard := resetTOTPGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", ActivateTOTP: true})

		if err := svc.RequestTOTPReset(ctx, user.Email); err != nil {
			t.Fatal(err)
		}

		if err := svc.ApproveTOTPResetRequest(ctx, user.ID); err != nil {
			t.Fatal(err)
		}

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.ResetTOTP(ctx, validGuard, user.ID, "password")
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPReset{Email: user.Email})

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
		if got := user.TOTPResetRequestedAt; !got.IsZero() {
			t.Errorf("want TOTP reset requested at to be zero; got %v", got)
		}
		if got := user.TOTPResetApprovedAt; !got.IsZero() {
			t.Error("want TOTP reset approved at to be cleared")
		}
		if want, got := 0, len(user.HashedRecoveryCodes); want != got {
			t.Error("want recovery codes to be cleared")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", VerifyTOTP: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", ActivateTOTP: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "qux@bar.com", ActivateTOTP: true})

		if err := svc.RequestTOTPReset(ctx, user2.Email); err != nil {
			t.Fatal(err)
		}

		if err := svc.ApproveTOTPResetRequest(ctx, user3.ID); err != nil {
			t.Fatal(err)
		}

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    resetTOTPGuard
			userID   string
			password string
			want     error
		}{
			{"invalid guard", invalidGuard, "", "", app.ErrForbidden},
			{"unactivated TOTP", validGuard, user1.ID, "password", nil},
			{"no reset requested approved", validGuard, user2.ID, "password", nil},
			{"incorrect password", validGuard, user3.ID, "123123123", nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ResetTOTP(ctx, tc.guard, tc.userID, tc.password)
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
