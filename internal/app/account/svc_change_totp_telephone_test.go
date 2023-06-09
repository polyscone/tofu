package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
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

	t.Run("success with unverified TOTP user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", SetupTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		newTel := errors.Must(account.NewTel("+81 70 0000 0000"))

		err := svc.ChangeTOTPTel(ctx, validGuard, user.ID, newTel.String())
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPTelChanged{
			Email:  user.Email,
			OldTel: user.TOTPTel,
			NewTel: newTel.String(),
		})

		user = errors.Must(store.FindUserByID(ctx, user.ID))

		if want, got := newTel.String(), user.TOTPTel; want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		err = svc.ChangeTOTPTel(ctx, validGuard, user.ID, user.TOTPTel)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Activate: true})
		user2 := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", VerifyTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name   string
			guard  changeTOTPTelGuard
			userID int
			newTel string
			want   error
		}{
			{"unauthorised", invalidGuard, 0, "", app.ErrUnauthorised},
			{"empty new phone", validGuard, user2.ID, "", app.ErrMalformedInput},
			{"new phone missing country code", validGuard, user2.ID, "070 0000 0001", app.ErrMalformedInput},
			{"activated user without TOTP setup", validGuard, user1.ID, "+81 70 0000 0003", app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ChangeTOTPTel(ctx, tc.guard, tc.userID, tc.newTel)
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

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", VerifyTOTP: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		oldTel := user.TOTPTel

		execute := func(newTel account.Tel) error {
			err := svc.ChangeTOTPTel(ctx, validGuard, user.ID, newTel.String())
			if err == nil {
				events.Expect(account.TOTPTelChanged{
					Email:  user.Email,
					OldTel: oldTel,
					NewTel: newTel.String(),
				})

				// Keep the old phone up to date with the latest phone
				// change so subsequent tests compare on the correct value
				oldTel = newTel.String()
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newTel account.Tel) bool {
				err := execute(newTel)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid new phone", func(t *testing.T) {
			quick.Check(t, func(newTel quick.Invalid[account.Tel]) bool {
				err := execute(newTel.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
