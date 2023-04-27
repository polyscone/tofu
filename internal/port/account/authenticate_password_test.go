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
	"github.com/polyscone/tofu/internal/port/account/domain"
	"github.com/polyscone/tofu/internal/port/account/repo/repotest"
)

func TestAuthenticateWithPassword(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db, []byte("s")))
	handler := account.NewAuthenticateWithPasswordHandler(broker, users)

	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", "password"))
	unactivatedUser := errors.Must(repotest.AddUser(t, users, ctx, "jane@doe.com", "password"))

	t.Run("success for matching email and password", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithPassword{
			Email:          activatedUser.Email.String(),
			IsAwaitingTOTP: false,
		})

		_, err := handler(ctx, account.AuthenticateWithPassword{
			Email:    activatedUser.Email.String(),
			Password: "password",
		})
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name     string
			email    string
			password string
			want     error
		}{
			{"empty email and password", "", "", port.ErrInvalidInput},
			{"empty email", "", "123", port.ErrInvalidInput},
			{"empty password", "empty@password.com", "", port.ErrInvalidInput},
			{"email without @ sign", "joebloggs.com", "password", port.ErrInvalidInput},
			{"non-existent email", "foo@bar.com", "password", nil},
			{"short password", "joe@bloggs.com", "0123456", port.ErrInvalidInput},
			{"incorrect password", activatedUser.Email.String(), "0123456789", port.ErrBadRequest},
			{"unactivated user", unactivatedUser.Email.String(), "password", port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				_, err := handler(ctx, account.AuthenticateWithPassword{
					Email:    tc.email,
					Password: tc.password,
				})
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("properties", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		execute := func(email text.Email, password domain.Password) error {
			_, err := handler(ctx, account.AuthenticateWithPassword{
				Email:    email.String(),
				Password: password.String(),
			})

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.CheckN(t, 2, func(email text.Email, password domain.Password) bool {
				err := execute(email, password)

				return !errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid email input", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email], password domain.Password) bool {
				err := execute(email.Unwrap(), password)

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid password input", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password quick.Invalid[domain.Password]) bool {
				err := execute(email, password.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
