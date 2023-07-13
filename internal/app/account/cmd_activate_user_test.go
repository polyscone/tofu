package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestActivateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com"})
		superRole := errsx.Must(repo.FindRoleByName(ctx, account.SuperRole.Name))

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		superUserCount := errsx.Must(repo.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 0, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}

		if err := svc.VerifyUser(ctx, user1.Email, "password", "password", account.VerifyUserOnly); err != nil {
			t.Fatal(err)
		}

		if err := svc.ActivateUser(ctx, user1.Email); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user1.Email})
		events.Expect(account.Activated{Email: user1.Email})
		events.Expect(account.RolesChanged{Email: user1.Email})

		user1 = errsx.Must(repo.FindUserByEmail(ctx, user1.Email))

		if user1.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}

		if err := svc.SignInWithPassword(ctx, user1.Email, "password"); err != nil {
			t.Errorf("want to be able to sign in with chosen password; got error: %v", err)
		}

		events.Expect(account.SignedInWithPassword{Email: user1.Email})

		superUserCount = errsx.Must(repo.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 1, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}

		if err := svc.VerifyUser(ctx, user2.Email, "password", "password", account.VerifyUserOnly); err != nil {
			t.Fatal(err)
		}

		if err := svc.ActivateUser(ctx, user2.Email); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user2.Email})
		events.Expect(account.Activated{Email: user2.Email})

		superUserCount = errsx.Must(repo.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 1, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}
	})

	t.Run("fail activating an already activated user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.ActivateUser(ctx, user.Email); err == nil {
			t.Error("want error; got <nil>")
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name         string
			email        string
			isValidInput bool
		}{
			{"valid inputs", "foo@example.com", true},

			{"invalid email empty", "", false},
			{"invalid email whitespace", "   ", false},
			{"invalid email missing @", "fooexample.com", false},
			{"invalid email part before @", "@example.com", false},
			{"invalid email part after @", "foo@", false},
			{"invalid email includes name", "Foo Bar <foo@example.com>", false},
			{"invalid email missing TLD", "foo@example.", false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ActivateUser(ctx, tc.email)
				switch {
				case err == nil:
					events.Expect(account.Activated{Email: tc.email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
