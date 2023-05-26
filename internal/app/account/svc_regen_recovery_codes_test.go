package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type regenerateRecoveryCodesGuard struct {
	value bool
}

func (g regenerateRecoveryCodesGuard) CanRegenerateRecoveryCodes(userID int) bool {
	return g.value
}

func TestRegenRecoveryCodes(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, VerifyTOTP: true})

	validGuard := regenerateRecoveryCodesGuard{value: true}
	invalidGuard := regenerateRecoveryCodesGuard{value: false}

	t.Run("success with user with verified TOTP", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.RecoveryCodesRegenerated{Email: verifiedTOTP.Email})

		originals := verifiedTOTP.RecoveryCodes

		err := svc.RegenerateRecoveryCodes(ctx, validGuard, verifiedTOTP.ID)
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, verifiedTOTP.ID))

		if len(user.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.RecoveryCodes {
				if len(rc.Code) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}

		if want, got := len(originals), len(user.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, original := range originals {
				for _, rc := range user.RecoveryCodes {
					if original.Code == rc.Code {
						t.Errorf("want different codes; got %q and %q", original.Code, rc.Code)
					}
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  account.RegenerateRecoveryCodesGuard
			userID int
			want   error
		}{
			{"unauthorised", invalidGuard, 0, app.ErrUnauthorised},
			{"TOTP not setup or verified", validGuard, activated.ID, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.RegenerateRecoveryCodes(ctx, tc.guard, tc.userID)
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
