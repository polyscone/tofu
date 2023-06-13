package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestSignUp(t *testing.T) {
	ctx := context.Background()

	t.Run("errors", func(t *testing.T) {
		svc, broker, repo := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if _, err := svc.SignUp(ctx, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedUp{Email: "foo@example.com"})

		if _, err := svc.SignUp(ctx, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedUp{Email: "foo@example.com"})

		user := MustAddUser(t, ctx, repo, TestUser{Email: "bar@example.com", Activate: true})
		_, err := svc.SignUp(ctx, user.Email)
		if want := app.ErrConflictingInput; !errors.Is(err, want) {
			t.Fatalf("want error: %v; got %v", want, err)
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
				_, err := svc.SignUp(ctx, tc.email)
				switch {
				case err == nil:
					events.Expect(account.SignedUp{Email: tc.email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
