package account_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/account/repotest"
)

func TestDisableTOTPRecoveryCode(t *testing.T) {
	validGuard := disableTOTPGuard{value: true}
	invalidGuard := disableTOTPGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewDisableTOTPRecoveryCodeHandler(broker, users)

	password := "password"
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))

	if err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}

	tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	_totp := errors.Must(tb.Generate(verifiedTOTPUser.TOTPKey, time.Now()))
	totp := errors.Must(account.NewTOTP(_totp))

	if err := verifiedTOTPUser.VerifyTOTP(totp, account.TOTPKindApp); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success with correct recovery code", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithRecoveryCode{
			Email: verifiedTOTPUser.Email.String(),
		})
		wantEvents = append(wantEvents, account.DisabledTOTP{
			Email: verifiedTOTPUser.Email.String(),
		})

		err := handler(ctx, account.DisableTOTPWithRecoveryCode{
			Guard:        validGuard,
			UserID:       verifiedTOTPUser.ID.String(),
			RecoveryCode: verifiedTOTPUser.RecoveryCodes[0].String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(users.FindByID(ctx, verifiedTOTPUser.ID))
		if want, got := []byte(nil), user.TOTPKey; !bytes.Equal(want, got) {
			t.Errorf("want %q; got %q", want, got)
		}
		if user.TOTPUseSMS {
			t.Error("want TOTP use SMS to be cleared")
		}
		if user.TOTPTelephone != "" {
			t.Error("want TOTP telephone to be cleared")
		}
		if user.TOTPKey != nil {
			t.Error("want TOTP key to be cleared")
		}
		if user.TOTPAlgorithm != "" {
			t.Error("want TOTP algorithm to be cleared")
		}
		if user.TOTPDigits != 0 {
			t.Error("want TOTP digits to be cleared")
		}
		if user.TOTPPeriod != 0 {
			t.Error("want TOTP period to be cleared")
		}
		if got := user.TOTPVerifiedAt; !got.IsZero() {
			t.Errorf("want TOTP verified at to be zero; got %v", got)
		}
		if want, got := 0, len(user.RecoveryCodes); want != got {
			t.Error("want recovery codes to be cleared")
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name         string
			guard        account.DisableTOTPGuard
			userID       string
			recoveryCode string
			want         error
		}{
			{"unauthorised", invalidGuard, "", "", port.ErrUnauthorised},
			{"incorrect recovery code", validGuard, verifiedTOTPUser.ID.String(), "ABCDEFGHIJKLM", port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.DisableTOTPWithRecoveryCode{
					Guard:        tc.guard,
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
}
