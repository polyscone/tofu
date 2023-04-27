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
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/repo/repotest"
)

func TestActivate(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db, []byte("s")))
	handler := account.NewActivateHandler(broker, users)

	user := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com", "password"))

	t.Run("success with existing user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.Activated{
			Email: user.Email.String(),
		})

		err := handler(ctx, account.Activate{
			Email: user.Email.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(users.FindByEmail(ctx, user.Email))

		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error activating an already activated user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		err := handler(ctx, account.Activate{
			Email: user.Email.String(),
		})
		if err == nil {
			t.Error("want error; got <nil>")
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("properties", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		execute := func(email text.Email) error {
			return handler(ctx, account.Activate{
				Email: email.String(),
			})
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email text.Email) bool {
				err := execute(email)

				return !errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email]) bool {
				err := execute(email.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
