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
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	user1Password := "password"
	user2Password := "password"
	user1 := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: user1Password, Activate: true})
	user2 := MustAddUser(t, ctx, store, TestUser{Email: "jane@doe.com", Password: user2Password, Activate: true})

	validGuard := changePasswordGuard{value: true}
	invalidGuard := changePasswordGuard{value: false}

	t.Run("success with activated logged in user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.PasswordChanged{Email: user1.Email})

		newPassword := errors.Must(account.NewPassword("password123"))
		err := svc.ChangePassword(ctx, validGuard, user1.ID, user1Password, newPassword.String(), newPassword.String())
		if err != nil {
			t.Fatal(err)
		}

		// Keep the old password up to date with the latest password
		// change so subsequent tests don't fail
		user1Password = newPassword.String()

		user := errors.Must(store.FindUserByID(ctx, user1.ID))

		if _, err := user.AuthenticateWithPassword(newPassword, hasher); err != nil {
			t.Errorf("want to be able to authenticate with new password; got %q", err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name        string
			guard       account.ChangePasswordGuard
			userID      int
			oldPassword string
			newPassword string
			want        error
		}{
			{"unauthorised", invalidGuard, 0, "", "", app.ErrUnauthorised},
			{"empty new password", validGuard, user2.ID, user2Password, "", app.ErrMalformedInput},
			{"empty old password", validGuard, user2.ID, "", user2Password, app.ErrMalformedInput},
			{"incorrect old password", validGuard, user2.ID, "password___", user2Password, app.ErrInvalidInput},
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
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(oldPassword, newPassword, newPasswordCheck account.Password) error {
			err := svc.ChangePassword(ctx, validGuard, user1.ID, oldPassword.String(), newPassword.String(), newPasswordCheck.String())
			if err == nil {
				events.Expect(account.PasswordChanged{Email: user1.Email})
			}

			return err
		}

		oldPassword := account.Password(user1Password)

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
