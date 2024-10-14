package account_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type regenerateRecoveryCodesGuard struct {
	value bool
}

func (g regenerateRecoveryCodesGuard) CanRegenerateRecoveryCodes(userID int) bool {
	return g.value
}

func TestRegenRecoveryCodes(t *testing.T) {
	validGuard := regenerateRecoveryCodesGuard{value: true}
	invalidGuard := regenerateRecoveryCodesGuard{value: false}

	t.Run("success with user with verified TOTP", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		originals := user.HashedRecoveryCodes

		totp := errsx.Must(user.GenerateTOTP())
		_, codes, err := svc.RegenerateRecoveryCodes(ctx, validGuard, user.ID, totp)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.RecoveryCodesRegenerated{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if want, got := len(codes), len(user.HashedRecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		}
		if len(user.HashedRecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.HashedRecoveryCodes {
				if len(rc) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}

		if want, got := len(originals), len(user.HashedRecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			for _, original := range originals {
				for _, rc := range user.HashedRecoveryCodes {
					if bytes.Equal(original, rc) {
						t.Errorf("want different codes; got %q and %q", original, rc)
					}
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com", Activate: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", ActivateTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			guard    regenerateRecoveryCodesGuard
			userID   int
			totpUser *account.User
			want     error
		}{
			{"invalid guard", invalidGuard, 0, user2, app.ErrForbidden},
			{"TOTP not setup or verified", validGuard, user1.ID, nil, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpUser != nil {
					totp = errsx.Must(tc.totpUser.GenerateTOTP())
				}

				_, _, err := svc.RegenerateRecoveryCodes(ctx, tc.guard, tc.userID, totp)
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
