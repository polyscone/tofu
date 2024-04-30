package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/testutil"
)

type activateTOTPGuard struct {
	value bool
}

func (g activateTOTPGuard) CanActivateTOTP(userID string) bool {
	return g.value
}

func TestActivateTOTP(t *testing.T) {
	validGuard := activateTOTPGuard{value: true}
	invalidGuard := activateTOTPGuard{value: false}

	t.Run("success with verified TOTP", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", VerifyTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.ActivateTOTP(ctx, validGuard, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if user.VerifiedAt.IsZero() {
			t.Error("want TOTP activated at to be populated; got zero")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", SetupTOTP: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "qux@quxx.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  activateTOTPGuard
			userID string
			want   error
		}{
			{"invalid guard", invalidGuard, "", app.ErrForbidden},
			{"unverified TOTP", validGuard, user1.ID, nil},
			{"already activated TOTP", validGuard, user2.ID, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ActivateTOTP(ctx, tc.guard, tc.userID)
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
