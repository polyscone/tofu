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

func TestSignInWithRecoveryCode(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	activatedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, ActivateTOTP: true})

	t.Run("success for valid user id and correct recovery code", func(t *testing.T) {
		t.Run("last signed in should not update on password auth", func(t *testing.T) {
			if !activatedTOTP.LastSignedInAt.IsZero() {
				t.Errorf("want last signed in at to be zero; got %v", activatedTOTP.LastSignedInAt)
			}

			err := svc.SignInWithPassword(ctx, activatedTOTP.Email, "password")
			if err != nil {
				t.Errorf("want <nil>; got %q", err)
			}

			if !activatedTOTP.LastSignedInAt.IsZero() {
				t.Errorf("want last signed in at to be zero; got %v", activatedTOTP.LastSignedInAt)
			}
		})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.SignedInWithRecoveryCode{Email: activatedTOTP.Email})

		nRecoveryCodes := len(activatedTOTP.RecoveryCodes)
		usedRecoveryCode := activatedTOTP.RecoveryCodes[0]

		err := svc.SignInWithRecoveryCode(ctx, activatedTOTP.ID, usedRecoveryCode.Code)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		activatedTOTP, err = store.FindUserByID(ctx, activatedTOTP.ID)
		if err != nil {
			t.Fatal(err)
		}

		if activatedTOTP.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}

		if want, got := nRecoveryCodes-1, len(activatedTOTP.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, rc := range activatedTOTP.RecoveryCodes {
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
			{"empty user id correct recovery code", 0, activatedTOTP.RecoveryCodes[1].Code, repo.ErrNotFound},
			{"empty user id incorrect recovery code", 0, incorrectCode, repo.ErrNotFound},
			{"activated user id incorrect recovery code", activatedTOTP.ID, incorrectCode, app.ErrInvalidInput},
			{"activated user id without TOTP setup", activated.ID, incorrectCode, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SignInWithRecoveryCode(ctx, tc.userID, tc.recoveryCode)
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
			err := svc.SignInWithRecoveryCode(ctx, activatedTOTP.ID, code.String())
			if err == nil {
				events.Expect(account.SignedInWithRecoveryCode{Email: activatedTOTP.Email})
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
