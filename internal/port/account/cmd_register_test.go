package account_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/repo"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	hasher := testutil.NewPasswordHasher()
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewRegisterHandler(broker, hasher, users)

	t.Run("properties", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		execute := func(id uuid.V4, email text.Email, password, passwordCheck account.Password) error {
			err := handler(ctx, account.Register{
				UserID:        id.String(),
				Email:         email.String(),
				Password:      password.String(),
				PasswordCheck: passwordCheck.String(),
			})
			if err == nil {
				wantEvents = append(wantEvents, account.Registered{
					Email: email.String(),
				})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(id uuid.V4, email text.Email, password account.Password) bool {
				if err := execute(id, email, password, password); err != nil {
					t.Log(err)

					return false
				}

				user := errors.Must(users.FindByEmail(ctx, email))

				return user.ActivatedAt.IsZero()
			})
		})

		t.Run("invalid id", func(t *testing.T) {
			quick.Check(t, func(id quick.Invalid[uuid.V4], email text.Email, password account.Password) bool {
				err := execute(id.Unwrap(), email, password, password)

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(id uuid.V4, email quick.Invalid[text.Email], password account.Password) bool {
				err := execute(id, email.Unwrap(), password, password)

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("invalid password", func(t *testing.T) {
			quick.Check(t, func(id uuid.V4, email text.Email, password quick.Invalid[account.Password]) bool {
				err := execute(id, email, password.Unwrap(), password.Unwrap())

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("mismatched password", func(t *testing.T) {
			quick.Check(t, func(id uuid.V4, email text.Email, password account.Password) bool {
				mismatch := bytes.Clone(password)
				mismatch[0]++

				err := execute(id, email, password, mismatch)

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
