package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
)

func TestSignUp(t *testing.T) {
	ctx := context.Background()

	t.Run("errors", func(t *testing.T) {
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if _, err := svc.SignUp(ctx, "foo@example.com"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.SignedUp{Email: "foo@example.com"})

		_, err := svc.SignUp(ctx, "foo@example.com")
		if want := app.ErrConflictingInput; !errors.Is(err, want) {
			t.Fatalf("want error: %T; got %T", want, err)
		}
	})

	t.Run("properties", func(t *testing.T) {
		svc, broker, store := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email account.Email, password, passwordCheck account.Password) error {
			_, err := svc.SignUp(ctx, email.String())
			if err == nil {
				events.Expect(account.SignedUp{Email: email.String()})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email account.Email, password account.Password) bool {
				if err := execute(email, password, password); err != nil {
					t.Log(err)

					return false
				}

				user := errors.Must(store.FindUserByEmail(ctx, email.String()))

				return !user.SignedUpAt.IsZero() && user.ActivatedAt.IsZero()
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[account.Email], password account.Password) bool {
				err := execute(email.Unwrap(), password, password)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
