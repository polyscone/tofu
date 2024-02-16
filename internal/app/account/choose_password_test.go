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

type choosePasswordGuard struct {
	value bool
}

func (g choosePasswordGuard) CanChoosePassword(userID int) bool {
	return g.value
}

func TestChoosePassword(t *testing.T) {
	validGuard := choosePasswordGuard{value: true}
	invalidGuard := choosePasswordGuard{value: false}

	t.Run("success with old password", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "joe@bloggs.com", VerifyNoPassword: true, Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		newPassword := errsx.Must(account.NewPassword("password123"))
		err := svc.ChoosePassword(ctx, validGuard, user.ID, newPassword.String(), newPassword.String())
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.PasswordChosen{Email: user.Email})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if len(user.HashedPassword) == 0 {
			t.Fatal("want user to have a hashed password")
		}

		if _, err := user.SignInWithPassword("site", newPassword, hasher); err != nil {
			t.Errorf("want to be able to sign in with new password; got %v", err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, repo, TestUser{Email: "jane@doe.com", VerifyNoPassword: true, Activate: true})
		user2 := MustAddUser(t, ctx, repo, TestUser{Email: "alan@doe.com", Verify: true, Activate: true})
		user3 := MustAddUser(t, ctx, repo, TestUser{Email: "bob@doe.com", VerifyNoPassword: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name        string
			guard       choosePasswordGuard
			userID      int
			newPassword string
			want        error
		}{
			{"invalid guard", invalidGuard, 0, "", app.ErrForbidden},
			{"empty new password", validGuard, user1.ID, "", app.ErrMalformedInput},
			{"already has a password", validGuard, user2.ID, "password123", nil},
			{"not activated", validGuard, user3.ID, "password123", nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ChoosePassword(ctx, tc.guard, tc.userID, tc.newPassword, tc.newPassword)
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

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name             string
			newPassword      string
			newPasswordCheck string
			isValidInput     bool
		}{
			{"valid inputs", "password1", "password1", true},

			{"invalid new password empty", "", "", false},
			{"invalid new password whitespace", "", "", false},
			{"invalid new password too short", ".......", ".......", false},
			{"invalid new password too long", strings.Repeat(".", 1001), strings.Repeat(".", 1001), false},

			{"invalid password check mismatch", "password1", "password2", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "foo@example.com", Activate: true})

				err := svc.ChoosePassword(ctx, validGuard, user.ID, tc.newPassword, tc.newPasswordCheck)
				switch {
				case err == nil:
					events.Expect(account.PasswordChosen{Email: user.Email})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
