package account_test

import (
	"bytes"
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

	password := "password"
	user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com"})

	t.Run("success with existing user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.Activated{Email: user.Email})
		events.Expect(account.SignedInWithPassword{Email: user.Email})

		if err := svc.ActivateUser(ctx, user.Email, password, password); err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByEmail(ctx, user.Email))

		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}

		if err := svc.SignInWithPassword(ctx, user.Email, password); err != nil {
			t.Errorf("want to be able to sign in with chosen password; got error: %v", err)
		}
	})

	t.Run("fail activating an already activated user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.ActivateUser(ctx, user.Email, password, password); err == nil {
			t.Error("want error; got <nil>")
		}
	})

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email text.Email, password, passwordCheck account.Password) error {
			err := svc.ActivateUser(ctx, email.String(), password.String(), passwordCheck.String())
			if err == nil {
				events.Expect(account.Activated{Email: email.String()})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password account.Password) bool {
				err := execute(email, password, password)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email], password account.Password) bool {
				err := execute(email.Unwrap(), password, password)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid password", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password quick.Invalid[account.Password]) bool {
				err := execute(email, password.Unwrap(), password.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("mismatched password", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password account.Password) bool {
				mismatch := bytes.Clone(password)
				mismatch[0]++

				err := execute(email, password, mismatch)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
