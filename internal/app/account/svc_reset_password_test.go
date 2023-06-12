package account_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

type resetPasswordGuard struct {
	value bool
}

func (g resetPasswordGuard) CanResetPassword(userID int) bool {
	return g.value
}

func TestResetPassword(t *testing.T) {
	validGuard := resetPasswordGuard{value: true}
	invalidGuard := resetPasswordGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		newPassword := errsx.Must(account.NewPassword("password123"))
		err := svc.ResetPassword(ctx, validGuard, user.ID, newPassword.String(), newPassword.String())
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.PasswordReset{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if _, err := user.SignInWithPassword(newPassword, hasher); err != nil {
			t.Errorf("want to be able to sign in with new password; got %q", err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name        string
			guard       resetPasswordGuard
			userID      int
			newPassword string
			want        error
		}{
			{"unauthorised", invalidGuard, 0, "", app.ErrUnauthorised},
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

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name             string
			newPassword      string
			newPasswordCheck string
			isValidInput     bool
		}{
			{"valid inputs", "password1", "password1", true},

			{"invalid password empty", "", "", false},
			{"invalid password whitespace", "        ", "        ", false},
			{"invalid password too short", ".......", ".......", false},
			{"invalid password too long", strings.Repeat(".", 101), strings.Repeat(".", 101), false},
			{"invalid password char NUL", "passwor\x00d", "passwor\x00d", false},
			{"invalid password char CR return", "passwor\rd", "passwor\rd", false},
			{"invalid password char LF", "passwor\nd", "passwor\nd", false},
			{"invalid password char tab", "passwor\td", "passwor\td", false},
			{"invalid password check mismatch", "password1", "password2", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "foo@example.com", Activate: true})

				err := svc.ResetPassword(ctx, validGuard, user.ID, tc.newPassword, tc.newPasswordCheck)
				switch {
				case err == nil:
					events.Expect(account.PasswordReset{Email: user.Email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
