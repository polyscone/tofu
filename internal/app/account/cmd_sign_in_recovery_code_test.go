package account_test

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/repository"
)

func TestSignInWithRecoveryCode(t *testing.T) {
	t.Run("success for valid user id and correct recovery code", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user, codes := MustAddUserRecoveryCodes(t, ctx, repo, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		err := svc.SignInWithPassword(ctx, user.Email, "password")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		nRecoveryCodes := len(user.HashedRecoveryCodes)
		usedRecoveryCodeHash := user.HashedRecoveryCodes[0]

		err = svc.SignInWithRecoveryCode(ctx, user.ID, codes[0])
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		events.Expect(account.SignedInWithRecoveryCode{Email: user.Email})

		user, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}

		if want, got := nRecoveryCodes-1, len(user.HashedRecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, rc := range user.HashedRecoveryCodes {
				if bytes.Equal(rc, usedRecoveryCodeHash) {
					t.Error("want used recovery code to be removed")
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com", Activate: true})
		user2, codes := MustAddUserRecoveryCodes(t, ctx, repo, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "bob@bloggs.com", Verify: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		incorrectCode := errsx.Must(account.NewRandomRecoveryCode()).String()

		tt := []struct {
			name         string
			userID       int
			recoveryCode string
			want         error
		}{
			{"empty user id correct recovery code", 0, codes[1], repository.ErrNotFound},
			{"empty user id incorrect recovery code", 0, incorrectCode, repository.ErrNotFound},
			{"activated user id without TOTP setup", user1.ID, incorrectCode, nil},
			{"activated user id incorrect recovery code", user2.ID, incorrectCode, app.ErrInvalidInput},
			{"unactivated user id", user3.ID, incorrectCode, account.ErrNotActivated},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SignInWithRecoveryCode(ctx, tc.userID, tc.recoveryCode)
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

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name         string
			code         string
			isValidInput bool
		}{
			{"valid inputs", "ABCDXYZ234567", true},

			{"invalid passcode empty", "", false},
			{"invalid passcode whitespace", "      ", false},
			{"invalid passcode lowercase letter", "a234567222222", false},
			{"invalid passcode invalid base32 number 1", "1234567222222", false},
			{"invalid passcode invalid base32 number 8", "8234567222222", false},
			{"invalid passcode invalid base32 number 9", "9234567222222", false},
			{"invalid passcode invalid base32 number 0", "0234567222222", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "joe@bloggs.com", ActivateTOTP: true})

				err := svc.SignInWithRecoveryCode(ctx, user.ID, tc.code)
				switch {
				case err == nil:
					events.Expect(account.SignedInWithRecoveryCode{Email: user.Email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
