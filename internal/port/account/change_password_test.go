package account_test

import (
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
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

type changePasswordGuard struct {
	canChangePassword bool
}

func (g changePasswordGuard) CanChangePassword(userID uuid.V4) bool {
	return g.canChangePassword
}

func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewChangePasswordHandler(broker, users)

	// Seed the repo
	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))

	t.Run("success with activated logged in user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.ChangedPassword{
			Email: activatedUser.Email.String(),
		})

		newPassword := errors.Must(domain.NewPassword("password123"))
		err := handler(ctx, account.ChangePassword{
			Guard:       changePasswordGuard{canChangePassword: true},
			UserID:      activatedUser.ID.String(),
			NewPassword: newPassword.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(users.FindByID(ctx, activatedUser.ID))

		if err := user.AuthenticateWithPassword(newPassword); err != nil {
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
			newPassword string
			want        error
		}{
			{"unauthorised", changePasswordGuard{canChangePassword: false}, activatedUser.ID.String(), "password123", port.ErrUnauthorised},
			{"empty password", changePasswordGuard{canChangePassword: true}, activatedUser.ID.String(), "", port.ErrInvalidInput},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.ChangePassword{
					Guard:       tc.guard,
					UserID:      tc.userID,
					NewPassword: tc.newPassword,
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

		execute := func(newPassword domain.Password) error {
			err := handler(ctx, account.ChangePassword{
				Guard:       changePasswordGuard{canChangePassword: true},
				UserID:      activatedUser.ID.String(),
				NewPassword: newPassword.String(),
			})
			if err == nil {
				wantEvents = append(wantEvents, account.ChangedPassword{
					Email: activatedUser.Email.String(),
				})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newPassword domain.Password) bool {
				err := execute(newPassword)

				return !errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid password", func(t *testing.T) {
			quick.Check(t, func(newPassword quick.Invalid[domain.Password]) bool {
				err := execute(newPassword.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
