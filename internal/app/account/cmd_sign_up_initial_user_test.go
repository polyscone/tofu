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

func TestSignUpInitialUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		password := errsx.Must(account.NewPassword("password123"))
		if err := svc.SignUpInitialUser(ctx, "foo@example.com", password.String(), password.String()); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.InitialUserSignedUp{
			Email:  "foo@example.com",
			System: "site",
			Method: account.SignUpMethodSystemSetup,
		})

		user := errsx.Must(repo.FindUserByEmail(ctx, "foo@example.com"))

		if len(user.HashedPassword) == 0 {
			t.Fatal("want user to have a hashed password")
		}
		if user.SignedUpAt.IsZero() {
			t.Error("want signed up at to be populated")
		}
		if want, got := "site", user.SignedUpSystem; want != got {
			t.Errorf("want signed up system to be %q; got %q", want, got)
		}
		if want, got := account.SignUpMethodSystemSetup, user.SignedUpMethod; want != got {
			t.Errorf("want signed up method to be %q; got %q", want, got)
		}
		if user.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}
		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}
		if !user.SignedUpAt.Equal(user.VerifiedAt) {
			t.Error("want signed up at and verified at to be the same")
		}
		if !user.VerifiedAt.Equal(user.ActivatedAt) {
			t.Error("want verified at and activated at to be the same")
		}
		if !user.IsSuper() {
			t.Error("want user to be assigned the super role")
		}

		if _, err := user.SignInWithPassword("site", password, hasher); err != nil {
			t.Errorf("want to be able to sign in with new password; got %q", err)
		}

		if err := svc.SignUpInitialUser(ctx, "bar@example.com", "password", "password"); err == nil {
			t.Error("want error after initial user created; got <nil>")
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnvWithSystem(ctx, "site")

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
				err := svc.SignUpInitialUser(ctx, tc.email, tc.password, tc.passwordCheck)
				switch {
				case err == nil:
					events.Expect(account.InitialUserSignedUp{
						Email:  tc.email,
						System: "site",
						Method: account.SignUpMethodSystemSetup,
					})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}