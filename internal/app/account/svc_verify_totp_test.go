package account_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/repository"
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

	t.Run("success with activated user with app TOTP setup", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", SetupTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		alg := errsx.Must(otp.NewAlgorithm(user.TOTPAlgorithm))
		tb := errsx.Must(otp.NewTimeBased(user.TOTPDigits, alg, time.Unix(0, 0), user.TOTPPeriod))
		totp := errsx.Must(tb.Generate(user.TOTPKey, time.Now()))

		// Deliberately set the method to SMS so we can check it's changed by the service
		user.TOTPMethod = account.TOTPMethodSMS.String()

		err := svc.VerifyTOTP(ctx, validGuard, user.ID, totp, account.TOTPMethodApp.String())
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if want, got := account.TOTPMethodApp.String(), user.TOTPMethod; want != got {
			t.Errorf("want TOTP method to be %q; got %q", want, got)
		}

		if len(user.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.RecoveryCodes {
				if len(rc) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}
	})

	t.Run("success with activated user with SMS TOTP setup", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "baz@qux.com", SetupTOTPTel: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		alg := errsx.Must(otp.NewAlgorithm(user.TOTPAlgorithm))
		tb := errsx.Must(otp.NewTimeBased(user.TOTPDigits, alg, time.Unix(0, 0), user.TOTPPeriod))
		totp := errsx.Must(tb.Generate(user.TOTPKey, time.Now()))

		// Deliberately set the method to app so we can check it's changed by the service
		user.TOTPMethod = account.TOTPMethodSMS.String()

		err := svc.VerifyTOTP(ctx, validGuard, user.ID, totp, account.TOTPMethodApp.String())
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if want, got := account.TOTPMethodApp.String(), user.TOTPMethod; want != got {
			t.Errorf("want TOTP method to be %q; got %q", want, got)
		}

		if len(user.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.RecoveryCodes {
				if len(rc) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Activate: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", SetupTOTP: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", ActivateTOTP: true})

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
			{"empty user id correct TOTP", validGuard, 0, user2, repository.ErrNotFound},
			{"empty user id incorrect TOTP", validGuard, 0, nil, repository.ErrNotFound},
			{"no TOTP user id correct TOTP", validGuard, user1.ID, user2, nil},
			{"already activated TOTP", validGuard, user3.ID, user3, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					alg := errsx.Must(otp.NewAlgorithm(tc.totpUser.TOTPAlgorithm))
					tb := errsx.Must(otp.NewTimeBased(tc.totpUser.TOTPDigits, alg, time.Unix(0, 0), tc.totpUser.TOTPPeriod))
					totp = errsx.Must(tb.Generate(tc.totpUser.TOTPKey, time.Now()))
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

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
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
			{"invalid passcode char NUL", "1234\x0056", false},
			{"invalid passcode char CR return", "1234\r56", false},
			{"invalid passcode char LF", "1234\n56", false},
			{"invalid passcode char tab", "1234\t56", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "joe@bloggs.com", ActivateTOTP: true})

				err := svc.VerifyTOTP(ctx, validGuard, user.ID, tc.totp, account.TOTPMethodApp.String())
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
