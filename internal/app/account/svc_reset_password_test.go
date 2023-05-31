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

type resetPasswordGuard struct {
	value bool
}

func (g resetPasswordGuard) CanResetPassword(userID int) bool {
	return g.value
}

func TestResetPassword(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	user := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})

	validGuard := resetPasswordGuard{value: true}
	invalidGuard := resetPasswordGuard{value: false}

	t.Run("success", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		events.Expect(account.PasswordReset{Email: user.Email})

		newPassword := errors.Must(account.NewPassword("password123"))
		err := svc.ResetPassword(ctx, validGuard, user.ID, newPassword.String(), newPassword.String())
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(store.FindUserByID(ctx, user.ID))

		if _, err := user.SignInWithPassword(newPassword, hasher); err != nil {
			t.Errorf("want to be able to sign in with new password; got %q", err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name        string
			guard       account.ResetPasswordGuard
			userID      int
			newPassword string
			want        error
		}{
			{"unauthorised", invalidGuard, 0, "", app.ErrUnauthorised},
			{"empty new password", validGuard, user.ID, "", app.ErrMalformedInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ResetPassword(ctx, tc.guard, tc.userID, tc.newPassword, tc.newPassword)
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

		execute := func(newPassword, newPasswordCheck account.Password) error {
			err := svc.ResetPassword(ctx, validGuard, user.ID, newPassword.String(), newPasswordCheck.String())
			if err == nil {
				events.Expect(account.PasswordReset{Email: user.Email})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				err := execute(newPassword, newPassword)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid new password", func(t *testing.T) {
			quick.Check(t, func(newPassword quick.Invalid[account.Password]) bool {
				err := execute(newPassword.Unwrap(), newPassword.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("mismatched password", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				mismatch := bytes.Clone(newPassword)
				mismatch[0]++

				err := execute(newPassword, mismatch)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
