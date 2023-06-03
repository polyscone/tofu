package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type activateTOTPGuard struct {
	value bool
}

func (g activateTOTPGuard) CanActivateTOTP(userID int) bool {
	return g.value
}

func TestActivateTOTP(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	unverifiedTOTPUser := MustAddUser(t, ctx, store, TestUser{Email: "jane@doe.com", Password: password, SetupTOTP: true})
	verifiedTOTPUser := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", Password: password, VerifyTOTP: true})
	activatedTOTPUser := MustAddUser(t, ctx, store, TestUser{Email: "qux@quxx.com", Password: password, ActivateTOTP: true})

	validGuard := activateTOTPGuard{value: true}
	invalidGuard := activateTOTPGuard{value: false}

	t.Run("success with verified TOTP", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.ActivateTOTP(ctx, validGuard, verifiedTOTPUser.ID)
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, verifiedTOTPUser.ID))

		if user.ActivatedAt.IsZero() {
			t.Error("want TOTP activated at to be populated; got zero")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  activateTOTPGuard
			userID int
			want   error
		}{
			{"unauthorised", invalidGuard, 0, app.ErrUnauthorised},
			{"unverified TOTP", validGuard, unverifiedTOTPUser.ID, app.ErrBadRequest},
			{"already activated TOTP", validGuard, activatedTOTPUser.ID, app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ActivateTOTP(ctx, tc.guard, tc.userID)
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
