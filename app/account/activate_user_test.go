package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type activateUsersGuard struct {
	value bool
}

func (g activateUsersGuard) CanActivateUsers() bool {
	return g.value
}

func TestActivateUser(t *testing.T) {
	validGuard := activateUsersGuard{value: true}
	invalidGuard := activateUsersGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com"})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		_, err := svc.VerifyUser(ctx, account.VerifyUserInput{
			Email:         user1.Email,
			Password:      "password",
			PasswordCheck: "password",
			Behavior:      account.VerifyUserOnly,
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := svc.ActivateUser(ctx, validGuard, user1.ID); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user1.Email})
		events.Expect(account.Activated{
			Email:       user1.Email,
			System:      "site",
			Method:      account.SignUpMethodWebForm,
			HasPassword: true,
		})

		user1 = errsx.Must(repo.FindUserByEmail(ctx, user1.Email))

		if user1.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}

		if _, err := svc.SignInWithPassword(ctx, user1.Email, "password"); err != nil {
			t.Errorf("want to be able to sign in with chosen password; got error: %v", err)
		}

		events.Expect(account.SignedIn{
			Email:  user1.Email,
			System: "site",
			Method: account.SignInMethodPassword,
		})

		_, err = svc.VerifyUser(ctx, account.VerifyUserInput{
			Email:         user2.Email,
			Password:      "password",
			PasswordCheck: "password",
			Behavior:      account.VerifyUserOnly,
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := svc.ActivateUser(ctx, validGuard, user2.ID); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user2.Email})
		events.Expect(account.Activated{
			Email:       user2.Email,
			System:      "site",
			Method:      account.SignUpMethodWebForm,
			HasPassword: true,
		})
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "john@doe.com"})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", Activate: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  activateUsersGuard
			userID int
			want   error
		}{
			{"invalid guard", invalidGuard, 0, app.ErrForbidden},
			{"user unverified", validGuard, user1.ID, account.ErrNotVerified},
			{"user already activated", validGuard, user2.ID, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.ActivateUser(ctx, tc.guard, tc.userID)
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
