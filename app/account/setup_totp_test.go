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

type setupTOTPGuard struct {
	value bool
}

func (g setupTOTPGuard) CanSetupTOTP(userID string) bool {
	return g.value
}

func TestSetupTOTP(t *testing.T) {
	validGuard := setupTOTPGuard{value: true}
	invalidGuard := setupTOTPGuard{value: false}

	t.Run("success with verified user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.SetupTOTP(ctx, validGuard, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

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

		err = svc.SetupTOTP(ctx, validGuard, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if bytes.Equal(oldTOTPKey, user.TOTPKey) {
			t.Error("want TOTP key to change")
		}
	})

	t.Run("success with verified user and verified TOTP", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "lisa@jones.com", VerifyTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.SetupTOTP(ctx, validGuard, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

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
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@bloggs.com", Verify: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  setupTOTPGuard
			userID string
			want   error
		}{
			{"invalid guard", invalidGuard, "", app.ErrForbidden},
			{"TOTP already setup and activated", validGuard, user1.ID, nil},
			{"unactivated user", validGuard, user2.ID, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SetupTOTP(ctx, tc.guard, tc.userID)
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
