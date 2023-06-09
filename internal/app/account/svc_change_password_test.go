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
)

type changePasswordGuard struct {
	value bool
}

func (g changePasswordGuard) CanChangePassword(userID int) bool {
	return g.value
}

func TestChangePassword(t *testing.T) {
	validGuard := changePasswordGuard{value: true}
	invalidGuard := changePasswordGuard{value: false}

	t.Run("success with activated", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		newPassword := errors.Must(account.NewPassword("password123"))
		err := svc.ChangePassword(ctx, validGuard, user.ID, "password", newPassword.String(), newPassword.String())
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.PasswordChanged{Email: user.Email})

		user = errors.Must(store.FindUserByID(ctx, user.ID))

		if _, err := user.SignInWithPassword(newPassword, hasher); err != nil {
			t.Errorf("want to be able to aign in with new password; got %q", err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "jane@doe.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name        string
			guard       changePasswordGuard
			userID      int
			oldPassword string
			newPassword string
			want        error
		}{
			{"unauthorised", invalidGuard, 0, "", "", app.ErrUnauthorised},
			{"empty new password", validGuard, user.ID, "password", "", app.ErrMalformedInput},
			{"empty old password", validGuard, user.ID, "", "password", app.ErrMalformedInput},
			{"incorrect old password", validGuard, user.ID, "password___", "password", app.ErrInvalidInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ChangePassword(ctx, tc.guard, tc.userID, tc.oldPassword, tc.newPassword, tc.newPassword)
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
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "jane@doe.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(oldPassword, newPassword, newPasswordCheck account.Password) error {
			err := svc.ChangePassword(ctx, validGuard, user.ID, oldPassword.String(), newPassword.String(), newPasswordCheck.String())
			if err == nil {
				events.Expect(account.PasswordChanged{Email: user.Email})
			}

			return err
		}

		oldPassword := account.Password("password")

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				err := execute(oldPassword, newPassword, newPassword)

				// Keep the old password up to date with the latest password
				// change so subsequent tests don't fail
				oldPassword = newPassword

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid old password", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				err := execute(newPassword, newPassword, newPassword)

				return errors.Is(err, app.ErrInvalidInput)
			})
		})

		t.Run("invalid new password", func(t *testing.T) {
			quick.Check(t, func(newPassword quick.Invalid[account.Password]) bool {
				err := execute(oldPassword, newPassword.Unwrap(), newPassword.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("mismatched password", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				mismatch := bytes.Clone(newPassword)
				mismatch[0]++

				err := execute(oldPassword, newPassword, mismatch)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
