package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

type changeTOTPTelephoneGuard struct {
	value bool
}

func (g changeTOTPTelephoneGuard) CanChangeTOTPTelephone(userID int) bool {
	return g.value
}

func TestChangeTOTPTelephone(t *testing.T) {
	ctx := context.Background()
	svc, broker, store := NewTestEnv(ctx)

	password := "password"
	activated := MustAddUser(t, ctx, store, TestUser{Email: "jim@bloggs.com", Password: password, Activate: true})
	unverifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com", Password: password, SetupTOTP: true})
	verifiedTOTP := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Password: password, VerifyTOTP: true})

	validGuard := changeTOTPTelephoneGuard{value: true}
	invalidGuard := changeTOTPTelephoneGuard{value: false}

	t.Run("success with unverified TOTP user", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		newTelephone := errors.Must(text.NewTel("+81 70 0000 0000"))

		err := svc.ChangeTOTPTelephone(ctx, validGuard, unverifiedTOTP.ID, newTelephone.String())
		if err != nil {
			t.Fatal(err)
		}

		events.Expect(account.TOTPTelephoneChanged{
			Email:        unverifiedTOTP.Email,
			OldTelephone: unverifiedTOTP.TOTPTelephone,
			NewTelephone: newTelephone.String(),
		})

		unverifiedTOTP = errors.Must(store.FindUserByID(ctx, unverifiedTOTP.ID))

		if want, got := newTelephone.String(), unverifiedTOTP.TOTPTelephone; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("no events when number is the same", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		err := svc.ChangeTOTPTelephone(ctx, validGuard, unverifiedTOTP.ID, unverifiedTOTP.TOTPTelephone)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name         string
			guard        changeTOTPTelephoneGuard
			userID       int
			newTelephone string
			want         error
		}{
			{"unauthorised", invalidGuard, 0, "", app.ErrUnauthorised},
			{"empty new telephone", validGuard, verifiedTOTP.ID, "", app.ErrMalformedInput},
			{"new telephone missing country code", validGuard, verifiedTOTP.ID, "070 0000 0001", app.ErrMalformedInput},
			{"activated user without TOTP setup", validGuard, activated.ID, "+81 70 0000 0003", app.ErrBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				err := svc.ChangeTOTPTelephone(ctx, tc.guard, tc.userID, tc.newTelephone)
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

		oldTelephone := verifiedTOTP.TOTPTelephone

		execute := func(newTelephone text.Tel) error {
			err := svc.ChangeTOTPTelephone(ctx, validGuard, verifiedTOTP.ID, newTelephone.String())
			if err == nil {
				events.Expect(account.TOTPTelephoneChanged{
					Email:        verifiedTOTP.Email,
					OldTelephone: oldTelephone,
					NewTelephone: newTelephone.String(),
				})

				// Keep the old telephone up to date with the latest telephone
				// change so subsequent tests compare on the correct value
				oldTelephone = newTelephone.String()
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newTelephone text.Tel) bool {
				err := execute(newTelephone)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid new telephone", func(t *testing.T) {
			quick.Check(t, func(newTelephone quick.Invalid[text.Tel]) bool {
				err := execute(newTelephone.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
