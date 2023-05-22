package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/account/repotest"
)

type changeTOTPTelephoneGuard struct {
	value bool
}

func (g changeTOTPTelephoneGuard) CanChangeTOTPTelephone(userID uuid.V4) bool {
	return g.value
}

func TestChangeTOTPTelephone(t *testing.T) {
	validGuard := changeTOTPTelephoneGuard{value: true}
	invalidGuard := changeTOTPTelephoneGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewChangeTOTPTelephoneHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	activatedNoTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jim@bloggs.com", password))
	unverifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "foo@bar.com", password))

	if err := activatedUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	activatedUser.TOTPVerifiedAt = time.Now()
	if err := users.Save(ctx, activatedUser); err != nil {
		t.Fatal(err)
	}

	if err := unverifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, unverifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success with unverified TOTP user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		newTelephone := errors.Must(text.NewTelephone("+81 70 0000 0000"))

		wantEvents = append(wantEvents, account.TOTPTelephoneChanged{
			Email:        unverifiedTOTPUser.Email.String(),
			OldTelephone: unverifiedTOTPUser.TOTPTelephone.String(),
			NewTelephone: newTelephone.String(),
		})

		err := handler(ctx, account.ChangeTOTPTelephone{
			Guard:        validGuard,
			UserID:       unverifiedTOTPUser.ID.String(),
			NewTelephone: newTelephone.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		unverifiedTOTPUser = errors.Must(users.FindByID(ctx, unverifiedTOTPUser.ID))

		if want, got := newTelephone, unverifiedTOTPUser.TOTPTelephone; want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)

		t.Run("no events when number is the same", func(t *testing.T) {
			wantEvents = nil
			gotEvents = nil

			err := handler(ctx, account.ChangeTOTPTelephone{
				Guard:        validGuard,
				UserID:       unverifiedTOTPUser.ID.String(),
				NewTelephone: unverifiedTOTPUser.TOTPTelephone.String(),
			})
			if err != nil {
				t.Fatal(err)
			}

			testutil.CheckEvents(t, wantEvents, gotEvents)
		})
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name         string
			guard        account.ChangeTOTPTelephoneGuard
			userID       string
			newTelephone string
			want         error
		}{
			{"unauthorised", invalidGuard, "", "", port.ErrUnauthorised},
			{"empty new telephone", validGuard, activatedUser.ID.String(), "", port.ErrMalformedInput},
			{"new telephone missing country code", validGuard, activatedUser.ID.String(), "070 0000 0001", port.ErrMalformedInput},
			{"activated user without TOTP setup", validGuard, activatedNoTOTPUser.ID.String(), "+81 70 0000 0003", port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.ChangeTOTPTelephone{
					Guard:        tc.guard,
					UserID:       tc.userID,
					NewTelephone: tc.newTelephone,
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

		oldTelephone := activatedUser.TOTPTelephone

		execute := func(newTelephone text.Telephone) error {
			err := handler(ctx, account.ChangeTOTPTelephone{
				Guard:        validGuard,
				UserID:       activatedUser.ID.String(),
				NewTelephone: newTelephone.String(),
			})
			if err == nil {
				wantEvents = append(wantEvents, account.TOTPTelephoneChanged{
					Email:        activatedUser.Email.String(),
					OldTelephone: oldTelephone.String(),
					NewTelephone: newTelephone.String(),
				})

				// Keep the old telephone up to date with the latest telephone
				// change so subsequent tests compare on the correct value
				oldTelephone = newTelephone
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(newTelephone text.Telephone) bool {
				err := execute(newTelephone)

				return !errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("invalid new telephone", func(t *testing.T) {
			quick.Check(t, func(newTelephone quick.Invalid[text.Telephone]) bool {
				err := execute(newTelephone.Unwrap())

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
