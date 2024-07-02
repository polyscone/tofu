package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/testx"
)

func TestRequestTOTPReset(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", ActivateTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		err := svc.RequestTOTPReset(ctx, user.Email)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPResetRequested{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))
		if got := user.TOTPResetRequestedAt; got.IsZero() {
			t.Error("want TOTP reset requested at to be populated; got zero")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", SetupTOTPTel: true, VerifyTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name  string
			email string
			want  error
		}{
			{"unactivated TOTP", user.Email, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.RequestTOTPReset(ctx, tc.email)
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
