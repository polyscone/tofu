package account_test

import (
	"bytes"
	"context"
	"sort"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type setupTOTPGuard struct {
	value bool
}

func (g setupTOTPGuard) CanSetupTOTP(userID int) bool {
	return g.value
}

func TestSetupTOTP(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, VerifyTOTP: true})

	validGuard := setupTOTPGuard{value: true}
	invalidGuard := setupTOTPGuard{value: false}

	t.Run("success with activated logged in user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.SetupTOTP(ctx, validGuard, activated.ID)
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, activated.ID))

		if len(user.TOTPKey) == 0 {
			t.Errorf("want TOTP key to be populated")
		}

		if len(user.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, rc := range user.RecoveryCodes {
				if len(rc.Code) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}

		oldTOTPKey := user.TOTPKey
		oldRecoveryCodes := user.RecoveryCodes

		err = svc.SetupTOTP(ctx, validGuard, activated.ID)
		if err != nil {
			t.Fatal(err)
		}

		user = errors.Must(store.FindUserByID(ctx, activated.ID))

		if want, got := oldTOTPKey, user.TOTPKey; !bytes.Equal(want, got) {
			t.Errorf("want TOTP key to remain unchanged (%q); got %q", want, got)
		}

		if want, got := len(oldRecoveryCodes), len(user.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			sort.Slice(oldRecoveryCodes, func(i, j int) bool { return oldRecoveryCodes[i].Code < oldRecoveryCodes[j].Code })
			sort.Slice(user.RecoveryCodes, func(i, j int) bool { return user.RecoveryCodes[i].Code < user.RecoveryCodes[j].Code })

			for i, rc := range oldRecoveryCodes {
				if want, got := rc.Code, user.RecoveryCodes[i].Code; want != got {
					t.Errorf("want code to remain unchanged (%q); got %q", want, got)
				}
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  account.SetupTOTPGuard
			userID int
			want   error
		}{
			{"unauthorised", invalidGuard, 0, app.ErrUnauthorised},
			{"TOTP already setup and verified", validGuard, verifiedTOTP.ID, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SetupTOTP(ctx, tc.guard, tc.userID)
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
