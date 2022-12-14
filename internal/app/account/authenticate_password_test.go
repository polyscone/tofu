package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/app/account/internal/repo/sqlite/repotest"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func TestAuthenticateWithPassword(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewAuthenticateWithPasswordHandler(broker, users)

	// Seed the repo
	verifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))
	errors.Must(repotest.AddUser(t, users, ctx, "jane@doe.com"))

	password := errors.Must(domain.NewPassword("password"))
	if err := verifiedUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success for matching email and password", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithPassword{
			Email:         verifiedUser.Email.String(),
			IsAwaitingMFA: false,
		})

		_, err := handler(ctx, account.AuthenticateWithPassword{
			Email:    "joe@bloggs.com",
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
			{"empty email and password", "", "", app.ErrInvalidInput},
			{"empty email", "", "123", app.ErrInvalidInput},
			{"empty password", "empty@password.com", "", app.ErrInvalidInput},
			{"email without @ sign", "joebloggs.com", "password", app.ErrInvalidInput},
			{"non-existent email", "foo@bar.com", "password", nil},
			{"short password", "joe@bloggs.com", "0123456", app.ErrInvalidInput},
			{"incorrect password", "joe@bloggs.com", "0123456789", app.ErrBadRequest},
			{"unverified user", "jane@doe.com", "password", app.ErrBadRequest},
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
			quick.Check(t, func(email text.Email, password domain.Password) bool {
				err := execute(email, password)

				return !errors.Is(err, app.ErrInvalidInput)
			})
		})

		t.Run("invalid email input", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email], password domain.Password) bool {
				err := execute(email.Unwrap(), password)

				return errors.Is(err, app.ErrInvalidInput)
			})
		})

		t.Run("invalid password input", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password quick.Invalid[domain.Password]) bool {
				err := execute(email, password.Unwrap())

				return errors.Is(err, app.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
