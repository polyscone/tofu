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
	t.Run("success for valid user id and correct recovery code", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})

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

		nRecoveryCodes := len(user.RecoveryCodes)
		usedRecoveryCode := user.RecoveryCodes[0]

		err = svc.SignInWithRecoveryCode(ctx, user.ID, usedRecoveryCode)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		events.Expect(account.SignedInWithRecoveryCode{Email: user.Email})

		user, err = store.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}

		if want, got := nRecoveryCodes-1, len(user.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, rc := range user.RecoveryCodes {
				if rc == usedRecoveryCode {
					t.Error("want used recovery code to be removed")
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Activate: true})
		user2 := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		incorrectCode := errors.Must(account.GenerateRecoveryCode()).String()

		tt := []struct {
			name         string
			userID       int
			recoveryCode string
			want         error
		}{
			{"empty user id correct recovery code", 0, user2.RecoveryCodes[1], repo.ErrNotFound},
			{"empty user id incorrect recovery code", 0, incorrectCode, repo.ErrNotFound},
			{"activated user id incorrect recovery code", user2.ID, incorrectCode, app.ErrInvalidInput},
			{"activated user id without TOTP setup", user1.ID, incorrectCode, app.ErrBadRequest},
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
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(code account.RecoveryCode) error {
			err := svc.SignInWithRecoveryCode(ctx, user.ID, code.String())
			if err == nil {
				events.Expect(account.SignedInWithRecoveryCode{Email: user.Email})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(code account.RecoveryCode) bool {
				err := execute(code)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid recovery code input", func(t *testing.T) {
			quick.Check(t, func(code quick.Invalid[account.RecoveryCode]) bool {
				err := execute(code.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
