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

type setupTOTPGuard struct {
	value bool
}

func (g setupTOTPGuard) CanSetupTOTP(userID int) bool {
	return g.value
}

func TestSetupTOTP(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "lisa@jones.com", Password: password, VerifyTOTP: true})
	activatedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, ActivateTOTP: true})

	validGuard := setupTOTPGuard{value: true}
	invalidGuard := setupTOTPGuard{value: false}

	t.Run("success with activated user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.SetupTOTP(ctx, validGuard, activated.ID)
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, activated.ID))

		if len(user.TOTPKey) == 0 {
			t.Error("want TOTP key to be populated")
		}
		if user.TOTPMethod != account.TOTPMethodNone.String() {
			t.Error("want TOTP method to be cleared")
		}
		if user.TOTPAlgorithm == "" {
			t.Error("want TOTP algorithm to be populated")
		}
		if user.TOTPDigits == 0 {
			t.Error("want TOTP digits to be set")
		}
		if user.TOTPPeriod == 0 {
			t.Error("want TOTP period to be set")
		}
		if got := user.TOTPVerifiedAt; !got.IsZero() {
			t.Errorf("want TOTP verified at to be zero; got %v", got)
		}
		if got := user.TOTPActivatedAt; !got.IsZero() {
			t.Errorf("want TOTP activated at to be zero; got %v", got)
		}

		oldTOTPKey := user.TOTPKey

		err = svc.SetupTOTP(ctx, validGuard, activated.ID)
		if err != nil {
			t.Fatal(err)
		}

		user = errors.Must(store.FindUserByID(ctx, activated.ID))

		if bytes.Equal(oldTOTPKey, user.TOTPKey) {
			t.Error("want TOTP key to change")
		}
	})

	t.Run("success with activated user and verified TOTP", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.SetupTOTP(ctx, validGuard, verifiedTOTP.ID)
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, verifiedTOTP.ID))

		if len(user.TOTPKey) == 0 {
			t.Error("want TOTP key to be populated")
		}
		if user.TOTPMethod != account.TOTPMethodNone.String() {
			t.Error("want TOTP method to be cleared")
		}
		if user.TOTPAlgorithm == "" {
			t.Error("want TOTP algorithm to be populated")
		}
		if user.TOTPDigits == 0 {
			t.Error("want TOTP digits to be set")
		}
		if user.TOTPPeriod == 0 {
			t.Error("want TOTP period to be set")
		}
		if got := user.TOTPVerifiedAt; !got.IsZero() {
			t.Errorf("want TOTP verified at to be zero; got %v", got)
		}
		if got := user.TOTPActivatedAt; !got.IsZero() {
			t.Errorf("want TOTP activated at to be zero; got %v", got)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  account.SetupTOTPGuard
			userID int
			want   error
		}{
			{"unauthorised", invalidGuard, 0, app.ErrUnauthorised},
			{"TOTP already setup and activated", validGuard, activatedTOTP.ID, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SetupTOTP(ctx, tc.guard, tc.userID)
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
