package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func TestActivateUser(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: "password"})

	t.Run("success with existing user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.Activated{Email: user.Email})

		if err := svc.ActivateUser(ctx, user.Email); err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByEmail(ctx, user.Email))

		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}
	})

	t.Run("fail activating an already activated user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.ActivateUser(ctx, user.Email); err == nil {
			t.Error("want error; got <nil>")
		}
	})

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email text.Email) error {
			err := svc.ActivateUser(ctx, email.String())
			if err == nil {
				events.Expect(account.Activated{Email: email.String()})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email text.Email) bool {
				err := execute(email)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email]) bool {
				err := execute(email.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
