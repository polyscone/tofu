package account_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestVerifyUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.VerifyUser(ctx, user.Email, "password", "password", account.VerifyUserOnly); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user.Email})

		user = errsx.Must(repo.FindUserByEmail(ctx, user.Email))

		if user.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}
	})

	t.Run("success and activate", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.VerifyUser(ctx, user.Email, "password", "password", account.VerifyUserActivate); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user.Email})
		events.Expect(account.Activated{Email: user.Email})

		user = errsx.Must(repo.FindUserByEmail(ctx, user.Email))

		if user.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}
		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}
	})

	t.Run("success invited always activate", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Invited: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.VerifyUser(ctx, user.Email, "password", "password", account.VerifyUserOnly); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user.Email})
		events.Expect(account.Activated{Email: user.Email})

		user = errsx.Must(repo.FindUserByEmail(ctx, user.Email))

		if user.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}
		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}
	})

	t.Run("fail verifying an already verified user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Verify: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.VerifyUser(ctx, user.Email, "password", "password", account.VerifyUserOnly); err == nil {
			t.Error("want error; got <nil>")
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name          string
			email         string
			password      string
			passwordCheck string
			isValidInput  bool
		}{
			{"valid inputs", "foo@example.com", "password", "password", true},

			{"invalid email empty", "", "password", "password", false},
			{"invalid email whitespace", "   ", "password", "password", false},
			{"invalid email missing @", "fooexample.com", "password", "password", false},
			{"invalid email part before @", "@example.com", "password", "password", false},
			{"invalid email part after @", "foo@", "password", "password", false},
			{"invalid email includes name", "Foo Bar <foo@example.com>", "password", "password", false},
			{"invalid email missing TLD", "foo@example.", "password", "password", false},

			{"invalid password empty", "foo@example.com", "", "", false},
			{"invalid password whitespace", "foo@example.com", "        ", "        ", false},
			{"invalid password too short", "foo@example.com", ".......", ".......", false},
			{"invalid password too long", "foo@example.com", strings.Repeat(".", 1001), strings.Repeat(".", 1001), false},
			{"invalid password check mismatch", "foo@example.com", "password", "password1", false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.VerifyUser(ctx, tc.email, tc.password, tc.passwordCheck, account.VerifyUserOnly)
				switch {
				case err == nil:
					events.Expect(account.Verified{Email: tc.email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
