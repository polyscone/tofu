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
)

func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	issuePassportHandler := account.NewIssuePassportHandler(broker, users)
	handler := account.NewChangePasswordHandler(broker, users)

	// Seed the repo
	verifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))
	unverifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "jane@doe.com"))

	password := errors.Must(domain.NewPassword("password"))
	if err := verifiedUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedUser); err != nil {
		t.Fatal(err)
	}

	verifiedUserLoggedOutPassport := errors.Must(issuePassportHandler(ctx, account.IssuePassport{
		UserID: verifiedUser.ID.String(),
	}))
	verifiedUserLoggedInPassport := errors.Must(issuePassportHandler(ctx, account.IssuePassport{
		UserID:     verifiedUser.ID.String(),
		IsLoggedIn: true,
	}))
	unverifiedUserPassport := errors.Must(issuePassportHandler(ctx, account.IssuePassport{
		UserID: unverifiedUser.ID.String(),
	}))

	t.Run("success with verified logged in user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.ChangedPassword{
			Email: verifiedUser.Email.String(),
		})

		newPassword := errors.Must(domain.NewPassword("password123"))
		err := handler(ctx, account.ChangePassword{
			Guard:       verifiedUserLoggedInPassport,
			UserID:      verifiedUserLoggedInPassport.UserID(),
			NewPassword: newPassword.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(users.FindByID(ctx, verifiedUser.ID))

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
			passport    account.Passport
			newPassword string
			want        error
		}{
			{"empty passport", account.EmptyPassport, "password123", app.ErrUnauthorized},
			{"empty password", verifiedUserLoggedInPassport, "", app.ErrInvalidInput},
			{"unverified user", unverifiedUserPassport, "password123", app.ErrUnauthorized},
			{"verified logged out user", verifiedUserLoggedOutPassport, "password123", app.ErrUnauthorized},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.ChangePassword{
					Guard:       tc.passport,
					UserID:      tc.passport.UserID(),
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
				Guard:       verifiedUserLoggedInPassport,
				UserID:      verifiedUserLoggedInPassport.UserID(),
				NewPassword: newPassword.String(),
			})
			if err == nil {
				wantEvents = append(wantEvents, account.ChangedPassword{
					Email: verifiedUser.Email.String(),
				})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newPassword domain.Password) bool {
				err := execute(newPassword)

				return !errors.Is(err, app.ErrInvalidInput)
			})
		})

		t.Run("invalid password", func(t *testing.T) {
			quick.Check(t, func(newPassword quick.Invalid[domain.Password]) bool {
				err := execute(newPassword.Unwrap())

				return errors.Is(err, app.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}