package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/pkg/errsx"
	"github.com/polyscone/tofu/pkg/testutil"
)

func TestDenyTOTPResetRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.RequestTOTPReset(ctx, user.Email)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPResetRequested{Email: user.Email})

		err = svc.DenyTOTPResetRequest(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPResetRequestDenied{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))
		if got := user.TOTPResetRequestedAt; !got.IsZero() {
			t.Error("want TOTP reset requested at to be cleared")
		}
	})
}