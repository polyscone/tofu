package account_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type verifyTOTPGuard struct {
	value bool
}

func (g verifyTOTPGuard) CanVerifyTOTP(userID int) bool {
	return g.value
}

func TestVerifyTOTP(t *testing.T) {
	validGuard := verifyTOTPGuard{value: true}
	invalidGuard := verifyTOTPGuard{value: false}

	t.Run("success with verified user with app TOTP setup", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", SetupTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		totp := errsx.Must(user.GenerateTOTP())

		// Deliberately set the method to SMS so we can check it's changed by the service
		user.TOTPMethod = account.TOTPMethodSMS.String()

		_, codes, err := svc.VerifyTOTP(ctx, validGuard, user.ID, totp, account.TOTPMethodApp.String())
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if want, got := account.TOTPMethodApp.String(), user.TOTPMethod; want != got {
			t.Errorf("want TOTP method to be %q; got %q", want, got)
		}

		if want, got := len(codes), len(user.HashedRecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		}
		if len(user.HashedRecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.HashedRecoveryCodes {
				if len(rc) == 0 {
					t.Fatal("want hashed code; got empty slice")
				}
			}
		}
	})

	t.Run("success with verified user with SMS TOTP setup", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "baz@qux.com", SetupTOTPTel: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		totp := errsx.Must(user.GenerateTOTP())

		// Deliberately set the method to app so we can check it's changed by the service
		user.TOTPMethod = account.TOTPMethodSMS.String()

		_, codes, err := svc.VerifyTOTP(ctx, validGuard, user.ID, totp, account.TOTPMethodApp.String())
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if want, got := account.TOTPMethodApp.String(), user.TOTPMethod; want != got {
			t.Errorf("want TOTP method to be %q; got %q", want, got)
		}

		if want, got := len(codes), len(user.HashedRecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		}
		if len(user.HashedRecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.HashedRecoveryCodes {
				if len(rc) == 0 {
					t.Fatal("want hashed code; got empty slice")
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Verify: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", SetupTOTP: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", ActivateTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    verifyTOTPGuard
			userID   int
			totpUser *account.User
			want     error
		}{
			{"invalid guard", invalidGuard, 0, nil, app.ErrForbidden},
			{"non-existent user id correct TOTP", validGuard, 999, user2, app.ErrNotFound},
			{"non-existent user id incorrect TOTP", validGuard, 999, nil, app.ErrNotFound},
			{"no TOTP user id correct TOTP", validGuard, user1.ID, user2, nil},
			{"already activated TOTP", validGuard, user3.ID, user3, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					totp = errsx.Must(tc.totpUser.GenerateTOTP())
				}

				_, _, err := svc.VerifyTOTP(ctx, tc.guard, tc.userID, totp, account.TOTPMethodApp.String())
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name         string
			totp         string
			isValidInput bool
		}{
			{"valid inputs", "123456", true},

			{"invalid passcode empty", "", false},
			{"invalid passcode whitespace", "      ", false},
			{"invalid passcode too short", "12345", false},
			{"invalid passcode too long", "1234567", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "joe@bloggs.com", ActivateTOTP: true})

				_, _, err := svc.VerifyTOTP(ctx, validGuard, user.ID, tc.totp, account.TOTPMethodApp.String())
				switch {
				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
