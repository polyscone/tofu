package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/testx"
)

func TestSignInWithFacebook(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", ActivateTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		_, signedIn, err := svc.SignInWithFacebook(ctx, user1.Email, account.FacebookSignInOnly)
		if err != nil {
			t.Fatal(err)
		}
		if !signedIn {
			t.Error("want signed in to be true; got false")
		}

		events.Expect(account.SignedIn{
			Email:  user1.Email,
			System: "site",
			Method: account.SignInMethodFacebook,
		})

		user1, err = repo.FindUserByID(ctx, user1.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user1.LastSignInAttemptAt.IsZero() {
			t.Error("want last sign in attempt at to be populated; got zero")
		}
		if want, got := "site", user1.LastSignInAttemptSystem; want != got {
			t.Errorf("want last sign in attempt system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodFacebook, user1.LastSignInAttemptMethod; want != got {
			t.Errorf("want last sign in attempt method to be %q; got %q", want, got)
		}
		if !user1.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be zero with TOTP")
		}
		if want, got := "", user1.LastSignedInSystem; want != got {
			t.Errorf("want last signed in system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodNone, user1.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}

		_, signedIn, err = svc.SignInWithFacebook(ctx, "bar@example.com", account.FacebookAllowSignUpActivate)
		if err != nil {
			t.Fatal(err)
		}
		if !signedIn {
			t.Error("want signed in to be true; got false")
		}

		events.Expect(account.SignedUp{
			Email:      "bar@example.com",
			System:     "site",
			Method:     account.SignUpMethodFacebook,
			IsVerified: true,
		})
		events.Expect(account.Activated{
			Email:       "bar@example.com",
			System:      "site",
			Method:      account.SignUpMethodFacebook,
			HasPassword: false,
		})
		events.Expect(account.SignedIn{
			Email:  "bar@example.com",
			System: "site",
			Method: account.SignInMethodFacebook,
		})

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
		if want, got := "site", user2.SignedUpSystem; want != got {
			t.Errorf("want signed up system to be %q; got %q", want, got)
		}
		if want, got := account.SignUpMethodFacebook, user2.SignedUpMethod; want != got {
			t.Errorf("want signed up method to be %q; got %q", want, got)
		}
		if !user2.LastSignedInAt.Equal(user2.LastSignInAttemptAt) {
			t.Error("want last signed in at to be the same as last sign in attempt at")
		}
		if want, got := "site", user2.LastSignedInSystem; want != got {
			t.Errorf("want last signed in system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodFacebook, user2.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}

		_, signedIn, err = svc.SignInWithFacebook(ctx, "bar@example.com", account.FacebookSignInOnly)
		if err != nil {
			t.Fatal(err)
		}
		if !signedIn {
			t.Error("want signed in to be true; got false")
		}

		events.Expect(account.SignedIn{
			Email:  user2.Email,
			System: "site",
			Method: account.SignInMethodFacebook,
		})

		user2, err = repo.FindUserByID(ctx, user2.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !user2.LastSignedInAt.Equal(user2.LastSignInAttemptAt) {
			t.Error("want last signed in at to be the same as last sign in attempt at")
		}
		if want, got := "site", user2.LastSignedInSystem; want != got {
			t.Errorf("want last signed in system to be %q; got %q", want, got)
		}
		if want, got := account.SignInMethodFacebook, user2.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@bloggs.com", Verify: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "john@doe.com", Suspend: true})
		user4 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", Verify: true, Suspend: true})
		user5 := MustAddUser(t, ctx, repo, TestUser{Email: "baz@qux.com", Activate: true, Suspend: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			email    string
			behavior account.FacebookSignInBehavior
			want     error
		}{
			{"empty email", "", account.FacebookSignInOnly, app.ErrMalformedInput},
			{"email without @ sign", "joebloggs.com", account.FacebookSignInOnly, app.ErrMalformedInput},
			{"sign in only with non-existent user", "foo+void@bar.com", account.FacebookSignInOnly, account.ErrFacebookSignUpDisabled},
			{"unverified", user1.Email, account.FacebookSignInOnly, account.ErrNotVerified},
			{"unactivated", user2.Email, account.FacebookSignInOnly, account.ErrNotActivated},
			{"unverified suspended user", user3.Email, account.FacebookSignInOnly, account.ErrNotVerified},
			{"unactivated suspended user", user4.Email, account.FacebookSignInOnly, account.ErrNotActivated},
			{"suspended user", user5.Email, account.FacebookSignInOnly, account.ErrSuspended},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, signedIn, err := svc.SignInWithFacebook(ctx, tc.email, tc.behavior)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
				if signedIn {
					t.Error("want signed in to be false; got true")
				}
			})
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

		MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", Verify: true})

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
				_, signedIn, err := svc.SignInWithFacebook(ctx, tc.email, account.FacebookAllowSignUpActivate)
				switch {
				case err == nil:
					events.Expect(account.SignedIn{
						Email:  tc.email,
						System: "site",
						Method: account.SignInMethodFacebook,
					})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
				if signedIn {
					t.Error("want signed in to be false; got true")
				}
			})
		}
	})
}
