package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/domain"
	"github.com/polyscone/tofu/internal/port/account/repo/repotest"
)

func TestAuthenticateWithRecoveryCode(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db, []byte("s")))
	handler := account.NewAuthenticateWithRecoveryCodeHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	activatedNoTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jim@bloggs.com", password))
	unverifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "foo@bar.com", password))

	if _, err := activatedUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := activatedUser.VerifyTOTPKey(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, activatedUser); err != nil {
		t.Fatal(err)
	}

	if _, err := unverifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, unverifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success for valid user id and correct recovery code", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithRecoveryCode{
			Email: activatedUser.Email.String(),
		})

		nRecoveryCodes := len(activatedUser.RecoveryCodes)
		usedRecoveryCode := activatedUser.RecoveryCodes[0].String()

		err := handler(ctx, account.AuthenticateWithRecoveryCode{
			UserID:       activatedUser.ID.String(),
			RecoveryCode: usedRecoveryCode,
		})
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)

		activatedUser, err = users.FindByID(ctx, activatedUser.ID)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := nRecoveryCodes-1, len(activatedUser.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		}
		if len(activatedUser.RecoveryCodes[0]) != 0 && usedRecoveryCode == activatedUser.RecoveryCodes[0].String() {
			t.Error("want used recovery code to be removed")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		incorrectCode := errors.Must(domain.GenerateRecoveryCode()).String()

		tt := []struct {
			name         string
			userID       string
			recoveryCode string
			want         error
		}{
			{"empty user id correct recovery code", "", string(activatedUser.RecoveryCodes[1]), port.ErrInvalidInput},
			{"empty user id incorrect recovery code", "", incorrectCode, port.ErrInvalidInput},
			{"activated user id incorrect recovery code", activatedUser.ID.String(), incorrectCode, port.ErrBadRequest},
			{"activated user id unverified correct TOTP", unverifiedTOTPUser.ID.String(), string(unverifiedTOTPUser.RecoveryCodes[0]), port.ErrBadRequest},
			{"activated user id without TOTP setup", activatedNoTOTPUser.ID.String(), incorrectCode, port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.AuthenticateWithRecoveryCode{
					UserID:       tc.userID,
					RecoveryCode: tc.recoveryCode,
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

		execute := func(code domain.RecoveryCode) error {
			err := handler(ctx, account.AuthenticateWithRecoveryCode{
				UserID:       activatedUser.ID.String(),
				RecoveryCode: code.String(),
			})

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(code domain.RecoveryCode) bool {
				err := execute(code)

				return !errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid recovery code input", func(t *testing.T) {
			quick.Check(t, func(code quick.Invalid[domain.RecoveryCode]) bool {
				err := execute(code.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
