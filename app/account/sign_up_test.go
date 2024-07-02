package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/testx"
)

func TestSignUp(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		if _, err := svc.SignUp(ctx, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedUp{
			Email:      "foo@example.com",
			System:     "site",
			Method:     account.SignUpMethodWebForm,
			IsVerified: false,
		})

		user, err := repo.FindUserByEmail(ctx, "foo@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if user.SignedUpAt.IsZero() {
			t.Error("want signed up at to be populated")
		}
		if want, got := "site", user.SignedUpSystem; want != got {
			t.Errorf("want signed up system to be %q; got %q", want, got)
		}
		if want, got := account.SignUpMethodWebForm, user.SignedUpMethod; want != got {
			t.Errorf("want signed up method to be %q; got %q", want, got)
		}

		if _, err := svc.SignUp(ctx, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedUp{
			Email:      "foo@example.com",
			System:     "site",
			Method:     account.SignUpMethodWebForm,
			IsVerified: false,
		})
	})

	t.Run("success already verified", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "pwa")

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", SignUpSystem: "site", Verify: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		if _, err := svc.SignUp(ctx, user.Email); err != nil {
			t.Fatal(err)
		}

		// Even though the user is already verified through the site we want the
		// event to record the current system they're using, which is "pwa" here
		events.Expect(account.SignedUp{
			Email:      user.Email,
			System:     "pwa",
			Method:     account.SignUpMethodWebForm,
			IsVerified: true,
		})

		user, err := repo.FindUserByEmail(ctx, user.Email)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "pwa", user.SignedUpSystem; want != got {
			t.Errorf("want signed up system to be changed to %q; got %q", want, got)
		}
	})

	t.Run("success already activated", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "pwa")

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", SignUpSystem: "site", Activate: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		if _, err := svc.SignUp(ctx, user.Email); err != nil {
			t.Fatal(err)
		}

		// Even though the user is already activated through the site we want the
		// event to record the current system they're using, which is "pwa" here
		events.Expect(account.AlreadySignedUp{
			Email:       user.Email,
			System:      "pwa",
			Method:      account.SignUpMethodWebForm,
			HasPassword: true,
		})

		user, err := repo.FindUserByEmail(ctx, user.Email)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "site", user.SignedUpSystem; want != got {
			t.Errorf("want signed up system to be unchanged from %q; got %q", want, got)
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
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
				_, err := svc.SignUp(ctx, tc.email)
				switch {
				case err == nil:
					user := errsx.Must(repo.FindUserByEmail(ctx, tc.email))

					events.Expect(account.SignedUp{
						Email:      tc.email,
						System:     user.SignedUpSystem,
						Method:     account.SignUpMethodWebForm,
						IsVerified: false,
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
