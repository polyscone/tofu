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

func TestRegisterUser(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email text.Email, password, passwordCheck account.Password) error {
			err := svc.RegisterUser(ctx, email.String(), password.String(), passwordCheck.String())
			if err == nil {
				events.Expect(account.Registered{Email: email.String()})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password account.Password) bool {
				if err := execute(email, password, password); err != nil {
					t.Log(err)

					return false
				}

				user := errors.Must(store.FindUserByEmail(ctx, email.String()))

				return !user.RegisteredAt.IsZero() && user.ActivatedAt.IsZero()
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