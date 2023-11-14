package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type inviteUserGuard struct {
	value bool
}

func (g inviteUserGuard) CanInviteUsers() bool {
	return g.value
}

func TestInviteUser(t *testing.T) {
	validGuard := inviteUserGuard{value: true}
	invalidGuard := inviteUserGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if _, err := svc.InviteUser(ctx, validGuard, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Invited{Email: "foo@example.com"})

		user, err := repo.FindUserByEmail(ctx, "foo@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if user.InvitedAt.IsZero() {
			t.Error("want invited at to be populated")
		}
		if !user.SignedUpAt.IsZero() {
			t.Error("want signed up at to be zero")
		}

		if _, err := svc.InviteUser(ctx, validGuard, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Invited{Email: "foo@example.com"})
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "foo@example.com", Verify: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "bar@example.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name  string
			guard inviteUserGuard
			email string
			want  error
		}{
			{"invalid guard", invalidGuard, "", app.ErrForbidden},
			{"verified user", validGuard, user1.Email, nil},
			{"activated user", validGuard, user2.Email, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.InviteUser(ctx, tc.guard, tc.email)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
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
				_, err := svc.InviteUser(ctx, validGuard, tc.email)
				switch {
				case err == nil:
					events.Expect(account.Invited{Email: tc.email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
