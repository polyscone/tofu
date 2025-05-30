package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

type unsuspendUserGuard struct {
	value bool
}

func (g unsuspendUserGuard) CanSuspendUsers() bool {
	return true
}

func (g unsuspendUserGuard) CanUnsuspendUsers() bool {
	return g.value
}

func TestUnsuspendUser(t *testing.T) {
	validGuard := unsuspendUserGuard{value: true}
	invalidGuard := unsuspendUserGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com"})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		if user.IsSuspended() {
			t.Error("want user to not be suspended")
		}

		// Unsuspending a user who isn't suspended shouldn't error or generate any events
		if _, err := svc.UnsuspendUser(ctx, validGuard, user.ID); err != nil {
			t.Fatal(err)
		}

		if _, err := svc.SuspendUser(ctx, validGuard, user.ID, "Foo bar baz"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Suspended{
			Email:  user.Email,
			Reason: "Foo bar baz",
		})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if !user.IsSuspended() {
			t.Error("want user to be suspended")
		}

		if want := "Foo bar baz"; user.SuspendedReason != want {
			t.Errorf("want suspended reason to be %q; got %q", want, user.SuspendedReason)
		}

		if _, err := svc.UnsuspendUser(ctx, validGuard, user.ID); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Unsuspended{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if user.IsSuspended() {
			t.Error("want user to not be suspended")
		}

		if want := ""; user.SuspendedReason != want {
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
			guard  unsuspendUserGuard
			userID int
			want   error
		}{
			{"invalid guard", invalidGuard, 0, app.ErrForbidden},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.UnsuspendUser(ctx, tc.guard, tc.userID)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
