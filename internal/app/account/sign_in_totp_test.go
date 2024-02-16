package account_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestSignInWithTOTP(t *testing.T) {
	t.Run("success for valid user id and correct totp", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

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

		if user.LastSignInAttemptAt.IsZero() {
			t.Error("want last sign in attempt at to be populated; got zero")
		}
		if want, got := "site", user.LastSignInAttemptSystem; want != got {
			t.Errorf("want last sign in attempt system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodPassword, user.LastSignInAttemptMethod; want != got {
			t.Errorf("want last sign in attempt method to be %q; got %q", want, got)
		}
		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}
		if want, got := "", user.LastSignedInSystem; want != got {
			t.Errorf("want last signed in system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodNone, user.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		totp := errsx.Must(user.GenerateTOTP())

		otp.CleanUsedTOTP(totp)

		err = svc.SignInWithTOTP(ctx, user.ID, totp)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		events.Expect(account.SignedIn{
			Email:  user.Email,
			System: "site",
			Method: account.SignInMethodPassword,
		})

		user, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !user.LastSignedInAt.Equal(user.LastSignInAttemptAt) {
			t.Error("want last signed in at to be the same as last sign in attempt at")
		}
		if want, got := "site", user.LastSignedInSystem; want != got {
			t.Errorf("want last signed in system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodPassword, user.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com", Verify: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", SetupTOTP: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", VerifyTOTP: true})
		user4 := MustAddUser(t, ctx, repo, TestUser{Email: "bob@jones.com", ActivateTOTP: true})
		user5 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@jones.com", Activate: true})
		user6 := MustAddUser(t, ctx, repo, TestUser{Email: "jim+suspended@bloggs.com", Verify: true, Suspend: true})
		user7 := MustAddUser(t, ctx, repo, TestUser{Email: "foo+suspended@bar.com", SetupTOTP: true, Suspend: true})
		user8 := MustAddUser(t, ctx, repo, TestUser{Email: "joe+suspended@bloggs.com", VerifyTOTP: true, Suspend: true})
		user9 := MustAddUser(t, ctx, repo, TestUser{Email: "bob+suspended@jones.com", ActivateTOTP: true, Suspend: true})
		user10 := MustAddUser(t, ctx, repo, TestUser{Email: "jane+suspended@jones.com", Activate: true, Suspend: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			userID   int
			totpUser *account.User
			want     error
		}{
			{"empty user id correct TOTP", 0, user4, app.ErrNotFound},
			{"empty user id incorrect TOTP", 0, nil, app.ErrNotFound},
			{"activated user id no TOTP", user5.ID, nil, nil},
			{"activated user id incorrect TOTP", user4.ID, nil, app.ErrInvalidInput},
			{"activated user id unverified correct TOTP", user2.ID, user2, nil},
			{"activated user id without TOTP activated", user3.ID, user3, nil},
			{"verified user id without TOTP setup", user1.ID, nil, account.ErrNotActivated},
			{"activated suspended user id no TOTP", user10.ID, nil, nil},
			{"activated suspended user id incorrect TOTP", user9.ID, nil, app.ErrInvalidInput},
			{"activated suspended user id unverified correct TOTP", user7.ID, user7, nil},
			{"activated suspended user id without TOTP activated", user8.ID, user8, nil},
			{"verified suspended user id without TOTP setup", user6.ID, nil, account.ErrNotActivated},
			{"activated suspended user id correct TOTP", user9.ID, user9, account.ErrSuspended},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					var err error
					totp, err = tc.totpUser.GenerateTOTP()
					if err != nil {
						t.Fatal(err)
					}
				}

				err := svc.SignInWithTOTP(ctx, tc.userID, totp)
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
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

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
					events.Expect(account.SignedIn{
						Email:  user.Email,
						System: "site",
						Method: account.SignInMethodPassword,
					})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
