package account_test

import (
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
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewRegisterHandler(broker, users)

	// Seed the repo
	errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))

	t.Run("properties", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		execute := func(id uuid.V4, email text.Email) error {
			err := handler(ctx, account.Register{
				UserID: id.String(),
				Email:  email.String(),
			})
			if err == nil {
				wantEvents = append(wantEvents, account.Registered{
					Email: email.String(),
				})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(id uuid.V4, email text.Email) bool {
				if err := execute(id, email); err != nil {
					t.Log(err)

					return false
				}

				user := errors.Must(users.FindByEmail(ctx, email))

				return user.ActivatedAt.IsZero()
			})
		})

		t.Run("invalid id", func(t *testing.T) {
			quick.Check(t, func(id quick.Invalid[uuid.V4], email text.Email) bool {
				err := execute(id.Unwrap(), email)

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(id uuid.V4, email quick.Invalid[text.Email]) bool {
				err := execute(id, email.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
