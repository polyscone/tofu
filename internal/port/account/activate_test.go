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
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

func TestActivate(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewActivateHandler(broker, users)

	user := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))

	t.Run("success with existing user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.Activated{
			Email: user.Email.String(),
		})

		err := handler(ctx, account.Activate{
			Email:    user.Email.String(),
			Password: "password",
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
			Email:    user.Email.String(),
			Password: "password",
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

		execute := func(email text.Email, password domain.Password) error {
			return handler(ctx, account.Activate{
				Email:    email.String(),
				Password: password.String(),
			})
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password domain.Password) bool {
				err := execute(email, password)

				return !errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email], password domain.Password) bool {
				err := execute(email.Unwrap(), password)

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid password", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password quick.Invalid[domain.Password]) bool {
				err := execute(email, password.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
