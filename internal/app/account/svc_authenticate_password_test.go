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

func TestAuthenticateWithPassword(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	unactivated := MustAddUser(t, ctx, store, TestUser{Email: "jane@doe.com", Password: "password"})
	activated := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: "password", Activate: true})

	t.Run("success for matching email and password", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.AuthenticatedWithPassword{Email: activated.Email})

		if !activated.LastSignedInAt.IsZero() {
			t.Errorf("want last signed in at to be zero; got %v", activated.LastSignedInAt)
		}

		err := svc.AuthenticateWithPassword(ctx, activated.Email, "password")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		activated, err = store.FindUserByID(ctx, activated.ID)
		if err != nil {
			t.Fatal(err)
		}

		if activated.LastSignedInAt.IsZero() {
			t.Error("want last signed in at to be populated; got zero")
		}
	})

	t.Run("failure cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name     string
			email    string
			password string
			want     error
		}{
			{"empty email and password", "", "", app.ErrMalformedInput},
			{"empty email", "", "123", app.ErrMalformedInput},
			{"empty password", "empty@password.com", "", app.ErrMalformedInput},
			{"email without @ sign", "joebloggs.com", "password", app.ErrMalformedInput},
			{"non-existent email", "foo@bar.com", "password", nil},
			{"short password", "joe@bloggs.com", "0123456", app.ErrMalformedInput},
			{"incorrect password", activated.Email, "0123456789", app.ErrBadRequest},
			{"unactivated user", unactivated.Email, "password", app.ErrBadRequest},
			{"unactivated user", unactivated.Email, "password", account.ErrNotActivated},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.AuthenticateWithPassword(ctx, tc.email, tc.password)
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want %q; got %q", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})

	t.Run("properties", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email text.Email, password account.Password) error {
			err := svc.AuthenticateWithPassword(ctx, email.String(), password.String())
			if err == nil {
				events.Expect(account.AuthenticatedWithPassword{Email: email.String()})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password account.Password) bool {
				err := execute(email, password)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid email input", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[text.Email], password account.Password) bool {
				err := execute(email.Unwrap(), password)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid password input", func(t *testing.T) {
			quick.Check(t, func(email text.Email, password quick.Invalid[account.Password]) bool {
				err := execute(email, password.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
