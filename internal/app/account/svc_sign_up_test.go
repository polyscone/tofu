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

func TestSignUp(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email text.Email, password, passwordCheck account.Password) error {
			_, err := svc.SignUp(ctx, email.String())
			if err == nil {
				events.Expect(account.SignedUp{Email: email.String()})
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

				return !user.SignedUpAt.IsZero() && user.ActivatedAt.IsZero()
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email], password account.Password) bool {
				err := execute(email.Unwrap(), password, password)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
