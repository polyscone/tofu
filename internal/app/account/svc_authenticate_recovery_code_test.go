package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/repo"
)

func TestAuthenticateWithRecoveryCode(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	unverifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", Password: password, SetupTOTP: true})
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, VerifyTOTP: true})

	t.Run("success for valid user id and correct recovery code", func(t *testing.T) {
		t.Run("last logged in should not update on password auth", func(t *testing.T) {
			if !verifiedTOTP.LastLoggedInAt.IsZero() {
				t.Errorf("want last logged in at to be zero; got %v", verifiedTOTP.LastLoggedInAt)
			}

			err := svc.AuthenticateWithPassword(ctx, verifiedTOTP.Email, "password")
			if err != nil {
				t.Errorf("want <nil>; got %q", err)
			}

			if !verifiedTOTP.LastLoggedInAt.IsZero() {
				t.Errorf("want last logged in at to be zero; got %v", verifiedTOTP.LastLoggedInAt)
			}
		})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.AuthenticatedWithRecoveryCode{Email: verifiedTOTP.Email})

		nRecoveryCodes := len(verifiedTOTP.RecoveryCodes)
		usedRecoveryCode := verifiedTOTP.RecoveryCodes[0]

		err := svc.AuthenticateWithRecoveryCode(ctx, verifiedTOTP.ID, usedRecoveryCode.Code)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		verifiedTOTP, err = store.FindUserByID(ctx, verifiedTOTP.ID)
		if err != nil {
			t.Fatal(err)
		}

		if verifiedTOTP.LastLoggedInAt.IsZero() {
			t.Error("want last logged in at to be populated; got zero")
		}

		if want, got := nRecoveryCodes-1, len(verifiedTOTP.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, rc := range verifiedTOTP.RecoveryCodes {
				if rc.Code == usedRecoveryCode.Code {
					t.Error("want used recovery code to be removed")
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		incorrectCode := errors.Must(account.GenerateCode()).String()

		tt := []struct {
			name         string
			userID       int
			recoveryCode string
			want         error
		}{
			{"empty user id correct recovery code", 0, verifiedTOTP.RecoveryCodes[1].Code, repo.ErrNotFound},
			{"empty user id incorrect recovery code", 0, incorrectCode, repo.ErrNotFound},
			{"activated user id incorrect recovery code", verifiedTOTP.ID, incorrectCode, app.ErrInvalidInput},
			{"activated user id unverified correct TOTP", unverifiedTOTP.ID, unverifiedTOTP.RecoveryCodes[0].Code, app.ErrBadRequest},
			{"activated user id without TOTP setup", activated.ID, incorrectCode, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.AuthenticateWithRecoveryCode(ctx, tc.userID, tc.recoveryCode)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(code account.Code) error {
			err := svc.AuthenticateWithRecoveryCode(ctx, verifiedTOTP.ID, code.String())
			if err == nil {
				events.Expect(account.AuthenticatedWithRecoveryCode{Email: verifiedTOTP.Email})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(code account.Code) bool {
				err := execute(code)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid recovery code input", func(t *testing.T) {
			quick.Check(t, func(code quick.Invalid[account.Code]) bool {
				err := execute(code.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
