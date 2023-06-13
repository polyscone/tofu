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

func TestSignInWithTOTP(t *testing.T) {
	t.Run("success for valid user id and correct totp", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "bob@jones.com", ActivateTOTP: true})

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		err := svc.SignInWithPassword(ctx, user.Email, "password")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		user, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		alg := errsx.Must(otp.NewAlgorithm(user.TOTPAlgorithm))
		tb := errsx.Must(otp.NewTimeBased(user.TOTPDigits, alg, time.Unix(0, 0), user.TOTPPeriod))
		totp := errsx.Must(tb.Generate(user.TOTPKey, time.Now()))

		otp.CleanUsedTOTP(totp)

		err = svc.SignInWithTOTP(ctx, user.ID, totp)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		events.Expect(account.SignedInWithTOTP{Email: user.Email})

		user, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com", Activate: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", SetupTOTP: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", VerifyTOTP: true})
		user4 := MustAddUser(t, ctx, repo, TestUser{Email: "bob@jones.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			userID   int
			totpUser *account.User
			want     error
		}{
			{"empty user id correct TOTP", 0, user4, repository.ErrNotFound},
			{"empty user id incorrect TOTP", 0, nil, repository.ErrNotFound},
			{"activated user id incorrect TOTP", user4.ID, nil, app.ErrInvalidInput},
			{"activated user id unverified correct TOTP", user2.ID, user2, nil},
			{"activated user id without TOTP setup", user1.ID, nil, nil},
			{"activated user id without TOTP activated", user3.ID, user3, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					alg := errsx.Must(otp.NewAlgorithm(tc.totpUser.TOTPAlgorithm))
					tb := errsx.Must(otp.NewTimeBased(tc.totpUser.TOTPDigits, alg, time.Unix(0, 0), tc.totpUser.TOTPPeriod))
					totp = errsx.Must(tb.Generate(tc.totpUser.TOTPKey, time.Now()))
				}

				err := svc.SignInWithTOTP(ctx, tc.userID, totp)
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
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "joe@bloggs.com", ActivateTOTP: true})

				err := svc.SignInWithTOTP(ctx, user.ID, tc.totp)
				switch {
				case err == nil:
					events.Expect(account.SignedInWithTOTP{Email: user.Email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
