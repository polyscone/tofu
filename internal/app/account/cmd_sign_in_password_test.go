package account_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestSignInWithPassword(t *testing.T) {
	t.Run("success for matching email and password", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if !user.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", user.LastSignedInAt)
		}

		err := svc.SignInWithPassword(ctx, user.Email, "password")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		events.Expect(account.SignedInWithPassword{Email: user.Email})

		user, err = repo.FindUserByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		if user.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
		if want, got := account.SignInMethodWebsite, user.LastSignedInMethod; want != got {
			t.Errorf("want last signed in method to be %q; got %q", want, got)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com"})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			email    string
			password string
			want     error
		}{
			{"empty email and password", "", "", app.ErrMalformedInput},
			{"empty email", "", "123", app.ErrMalformedInput},
			{"empty password", "empty@password.com", "", app.ErrMalformedInput},
			{"email without @ sign", "joebloggs.com", "password", app.ErrMalformedInput},
			{"non-existent email", "foo@bar.com", "password", nil},
			{"short password", "joe@bloggs.com", "0123456", app.ErrMalformedInput},
			{"incorrect password", user2.Email, "0123456789", app.ErrBadRequest},
			{"unactivated user bad request", user1.Email, "password", app.ErrBadRequest},
			{"unactivated user", user1.Email, "password", account.ErrNotActivated},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SignInWithPassword(ctx, tc.email, tc.password)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
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
			password     string
			isValidInput bool
		}{
			{"valid inputs", "foo@example.com", "password", true},

			{"invalid email empty", "", "password", false},
			{"invalid email whitespace", "   ", "password", false},
			{"invalid email missing @", "fooexample.com", "password", false},
			{"invalid email part before @", "@example.com", "password", false},
			{"invalid email part after @", "foo@", "password", false},
			{"invalid email includes name", "Foo Bar <foo@example.com>", "password", false},
			{"invalid email missing TLD", "foo@example.", "password", false},
			{"invalid email char NUL", "foo\x00@example.com", "password", false},
			{"invalid email char CR return", "foo\r@example.com", "password", false},
			{"invalid email char LF", "foo\n@example.com", "password", false},
			{"invalid email char tab", "foo\t@example.com", "password", false},

			{"invalid password empty", "foo@example.com", "", false},
			{"invalid password whitespace", "foo@example.com", "        ", false},
			{"invalid password too short", "foo@example.com", ".......", false},
			{"invalid password too long", "foo@example.com", strings.Repeat(".", 101), false},
			{"invalid password char NUL", "foo@example.com", "passwor\x00d", false},
			{"invalid password char CR return", "foo@example.com", "passwor\rd", false},
			{"invalid password char LF", "foo@example.com", "passwor\nd", false},
			{"invalid password char tab", "foo@example.com", "passwor\td", false},
			{"invalid password check mismatch", "foo@example.com", "password", false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.SignInWithPassword(ctx, tc.email, tc.password)
				switch {
				case err == nil:
					events.Expect(account.SignedInWithPassword{Email: tc.email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
