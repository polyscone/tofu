package account_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
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

	t.Run("success with old password", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", Activate: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		newPassword := errsx.Must(account.NewPassword("password123"))
		_, err := svc.ChangePassword(ctx, validGuard, account.ChangePasswordInput{
			UserID:           user.ID,
			OldPassword:      "password",
			NewPassword:      newPassword.String(),
			NewPasswordCheck: newPassword.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.PasswordChanged{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if len(user.HashedPassword) == 0 {
			t.Fatal("want user to have a hashed password")
		}

		if _, err := user.SignInWithPassword("site", newPassword, hasher); err != nil {
			t.Errorf("want to be able to sign in with new password; got %q", err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", Activate: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name        string
			guard       changePasswordGuard
			userID      int
			oldPassword string
			newPassword string
			want        error
		}{
			{"invalid guard", invalidGuard, 0, "", "", app.ErrForbidden},
			{"empty new password", validGuard, user.ID, "password", "", app.ErrMalformedInput},
			{"empty old password", validGuard, user.ID, "", "password", app.ErrMalformedInput},
			{"incorrect old password", validGuard, user.ID, "password___", "password", app.ErrInvalidInput},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.ChangePassword(ctx, tc.guard, account.ChangePasswordInput{
					UserID:           tc.userID,
					OldPassword:      tc.oldPassword,
					NewPassword:      tc.newPassword,
					NewPasswordCheck: tc.newPassword,
				})
				switch {
				case tc.want != nil && !errors.Is(err, tc.want):
					t.Errorf("want error: %v; got: %v", tc.want, err)

				case err == nil:
					t.Error("want error; got <nil>")
				}
			})
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name             string
			oldPassword      string
			newPassword      string
			newPasswordCheck string
			isValidInput     bool
		}{
			{"valid inputs", "password", "password1", "password1", true},

			{"invalid old password empty", "", "password1", "password1", false},
			{"invalid old password whitespace", "        ", "password1", "password1", false},
			{"invalid old password too short", ".......", "password1", "password1", false},
			{"invalid old password too long", strings.Repeat(".", 1001), "password1", "password1", false},

			{"invalid new password empty", "", "", "", false},
			{"invalid new password whitespace", "        ", "", "", false},
			{"invalid new password too short", "password", ".......", ".......", false},
			{"invalid new password too long", "password", strings.Repeat(".", 1001), strings.Repeat(".", 1001), false},

			{"invalid password check mismatch", "password", "password1", "password2", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "foo@example.com", Activate: true})

				_, err := svc.ChangePassword(ctx, validGuard, account.ChangePasswordInput{
					UserID:           user.ID,
					OldPassword:      tc.oldPassword,
					NewPassword:      tc.newPassword,
					NewPasswordCheck: tc.newPasswordCheck,
				})
				switch {
				case err == nil:
					events.Expect(account.PasswordChanged{Email: user.Email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
