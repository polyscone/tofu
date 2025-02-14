package account_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type suspendUserGuard struct {
	value bool
}

func (g suspendUserGuard) CanSuspendUsers() bool {
	return g.value
}

func (g suspendUserGuard) CanChangeRoles(userID int) bool {
	return true
}

func TestSuspendUser(t *testing.T) {
	validGuard := suspendUserGuard{value: true}
	invalidGuard := suspendUserGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		if user.IsSuspended() {
			t.Error("want user to not be suspended")
		}

		if _, err := svc.SuspendUser(ctx, validGuard, user.ID, "Foo bar baz"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Suspended{
			Email:  user.Email,
			Reason: "Foo bar baz",
		})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		suspendedAt := user.SuspendedAt
		if !user.IsSuspended() {
			t.Error("want user to be suspended")
		}

		if want := "Foo bar baz"; user.SuspendedReason != want {
			t.Errorf("want suspended reason to be %q; got %q", want, user.SuspendedReason)
		}

		if _, err := svc.SuspendUser(ctx, validGuard, user.ID, "Qux"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SuspendedReasonChanged{
			Email:  user.Email,
			Reason: "Qux",
		})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if !user.IsSuspended() {
			t.Error("want user to be suspended")
		}
		if !user.SuspendedAt.Equal(suspendedAt) {
			t.Error("want a second suspension to not change the earlier suspension time")
		}

		if want := "Qux"; user.SuspendedReason != want {
			t.Errorf("want suspended reason to be %q; got %q", want, user.SuspendedReason)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  suspendUserGuard
			userID int
			want   error
		}{
			{"invalid guard", invalidGuard, 0, app.ErrForbidden},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.SuspendUser(ctx, tc.guard, tc.userID, "")
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
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "john@doe.com"})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name            string
			suspendedReason string
			isValidInput    bool
		}{
			{"valid suspended reason empty", "", true},
			{"valid suspended reason populated", "Foo bar baz qux...", true},

			{"invalid suspended reason too long", strings.Repeat(".", 101), false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.SuspendUser(ctx, validGuard, user.ID, tc.suspendedReason)
				switch {
				case err == nil:
					if user.IsSuspended() {
						events.Expect(account.SuspendedReasonChanged{
							Email:  user.Email,
							Reason: tc.suspendedReason,
						})
					} else {
						events.Expect(account.Suspended{
							Email:  user.Email,
							Reason: tc.suspendedReason,
						})
					}

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}

				user = errsx.Must(repo.FindUserByID(ctx, user.ID))
			})
		}
	})
}
