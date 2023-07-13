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

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com"})
		superRole := errsx.Must(repo.FindRoleByName(ctx, account.SuperRole.Name))

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		superUserCount := errsx.Must(repo.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 0, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}

		if err := svc.VerifyUser(ctx, user1.Email, "password", "password"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user1.Email})
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

		if err := svc.VerifyUser(ctx, user2.Email, "password", "password"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Verified{Email: user2.Email})

		superUserCount = errsx.Must(repo.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 1, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}
	})

	t.Run("fail activating an already verified user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Verify: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.VerifyUser(ctx, user.Email, "password", "password"); err == nil {
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
				err := svc.VerifyUser(ctx, tc.email, tc.password, tc.passwordCheck)
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
