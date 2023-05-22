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
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/account/repotest"
)

type changePasswordGuard struct {
	value bool
}

func (g changePasswordGuard) CanChangePassword(userID uuid.V4) bool {
	return g.value
}

func TestChangePassword(t *testing.T) {
	validGuard := changePasswordGuard{value: true}
	invalidGuard := changePasswordGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	hasher := testutil.NewPasswordHasher()
	handler := account.NewChangePasswordHandler(broker, hasher, users)

	user1Password := "password"
	user2Password := "password"
	user1 := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", user1Password))
	user2 := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", user2Password))

	t.Run("success with activated logged in user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.PasswordChanged{
			Email: user1.Email.String(),
		})

		newPassword := errors.Must(account.NewPassword("password123"))
		err := handler(ctx, account.ChangePassword{
			Guard:            validGuard,
			UserID:           user1.ID.String(),
			OldPassword:      user1Password,
			NewPassword:      newPassword.String(),
			NewPasswordCheck: newPassword.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Keep the old password up to date with the latest password
		// change so subsequent tests don't fail
		user1Password = newPassword.String()

		user := errors.Must(users.FindByID(ctx, user1.ID))

		if _, err := user.AuthenticateWithPassword(newPassword, hasher); err != nil {
			t.Errorf("want to be able to authenticate with new password; got %q", err)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name        string
			guard       account.ChangePasswordGuard
			userID      string
			oldPassword string
			newPassword string
			want        error
		}{
			{"unauthorised", invalidGuard, "", "", "", port.ErrUnauthorised},
			{"empty new password", validGuard, user2.ID.String(), user2Password, "", port.ErrMalformedInput},
			{"empty old password", validGuard, user2.ID.String(), "", user2Password, port.ErrMalformedInput},
			{"incorrect old password", validGuard, user2.ID.String(), "password___", user2Password, port.ErrInvalidInput},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.ChangePassword{
					Guard:            tc.guard,
					UserID:           tc.userID,
					OldPassword:      tc.oldPassword,
					NewPassword:      tc.newPassword,
					NewPasswordCheck: tc.newPassword,
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

		execute := func(oldPassword, newPassword, newPasswordCheck account.Password) error {
			err := handler(ctx, account.ChangePassword{
				Guard:            validGuard,
				UserID:           user1.ID.String(),
				OldPassword:      oldPassword.String(),
				NewPassword:      newPassword.String(),
				NewPasswordCheck: newPasswordCheck.String(),
			})
			if err == nil {
				wantEvents = append(wantEvents, account.PasswordChanged{
					Email: user1.Email.String(),
				})
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

				return !errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("invalid old password", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				err := execute(newPassword, newPassword, newPassword)

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid new password", func(t *testing.T) {
			quick.Check(t, func(newPassword quick.Invalid[account.Password]) bool {
				err := execute(oldPassword, newPassword.Unwrap(), newPassword.Unwrap())

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("mismatched password", func(t *testing.T) {
			quick.Check(t, func(newPassword account.Password) bool {
				mismatch := bytes.Clone(newPassword)
				mismatch[0]++

				err := execute(oldPassword, newPassword, mismatch)

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
