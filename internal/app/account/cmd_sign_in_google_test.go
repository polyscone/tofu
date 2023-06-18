package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestSignInWithGoogle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", ActivateTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.SignInWithGoogle(ctx, user1.Email)
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedInWithGoogle{Email: user1.Email})

		user1, err = repo.FindUserByID(ctx, user1.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !user1.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be zero with TOTP")
		}
		if want, got := account.SignInMethodNone, user1.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}

		err = svc.SignInWithGoogle(ctx, "bar@example.com")
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedUpWithGoogle{Email: "bar@example.com"})
		events.Expect(account.SignedInWithGoogle{Email: "bar@example.com"})

		user2, err := repo.FindUserByEmail(ctx, "bar@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if len(user2.HashedPassword) != 0 {
			t.Error("want hashed password to be empty")
		}
		if user2.ActivatedAt.IsZero() {
			t.Error("want activated at to be populated")
		}
		if user2.SignedUpAt.IsZero() {
			t.Error("want signed up at to be populated")
		}
		if user2.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
		if want, got := account.SignInMethodGoogle, user2.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}

		err = svc.SignInWithGoogle(ctx, "bar@example.com")
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedInWithGoogle{Email: user2.Email})

		user2, err = repo.FindUserByID(ctx, user2.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user2.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
		if want, got := account.SignInMethodGoogle, user2.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", Activate: true})

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
				err := svc.SignInWithGoogle(ctx, tc.email)
				switch {
				case err == nil:
					events.Expect(account.SignedInWithGoogle{Email: tc.email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
