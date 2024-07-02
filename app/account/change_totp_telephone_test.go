package account_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/testx"
)

type changeTOTPTelGuard struct {
	value bool
}

func (g changeTOTPTelGuard) CanChangeTOTPTel(userID int) bool {
	return g.value
}

func TestChangeTOTPTel(t *testing.T) {
	validGuard := changeTOTPTelGuard{value: true}
	invalidGuard := changeTOTPTelGuard{value: false}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "foo@bar.com", SetupTOTP: true})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		newTel := errsx.Must(account.NewTel("+81 70 0000 0000"))

		_, err := svc.ChangeTOTPTel(ctx, validGuard, user.ID, newTel.String())
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPTelChanged{
			Email:  user.Email,
			OldTel: user.TOTPTel,
			NewTel: newTel.String(),
		})

		user = errsx.Must(repo.FindUserByID(ctx, user.ID))

		if want, got := newTel.String(), user.TOTPTel; want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		_, err = svc.ChangeTOTPTel(ctx, validGuard, user.ID, user.TOTPTel)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, repo, TestUser{Email: "jim@bloggs.com"})

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  changeTOTPTelGuard
			userID int
			newTel string
			want   error
		}{
			{"invalid guard", invalidGuard, 0, "", app.ErrForbidden},
			{"user without TOTP setup", validGuard, user.ID, "+81 70 0000 0003", nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.ChangeTOTPTel(ctx, tc.guard, tc.userID, tc.newTel)
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
			name         string
			newTel       string
			isValidInput bool
		}{
			{"valid inputs", "+81 12 3456 7890", true},

			{"invalid empty", "", false},
			{"invalid whitespace", "   ", false},
			{"invalid missing country code", "081 12 3456 7890", false},
			{"invalid contains hyphens", "+81-12-3456-7890", false},
			{"invalid contains letters", "+81a12b3456c7890", false},
		}
		for i, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				user := MustAddUser(t, ctx, repo, TestUser{Email: strconv.Itoa(i) + "foo@example.com", SetupTOTPTel: true, VerifyTOTP: true})

				_, err := svc.ChangeTOTPTel(ctx, validGuard, user.ID, tc.newTel)
				switch {
				case err == nil:
					events.Expect(account.TOTPTelChanged{
						Email:  user.Email,
						OldTel: "+00 00 0000 0000",
						NewTel: tc.newTel,
					})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
