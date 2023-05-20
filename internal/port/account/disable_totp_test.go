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
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/domain"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/repotest"
)

type disableTOTPGuard struct {
	value bool
}

func (g disableTOTPGuard) CanDisableTOTP(userID uuid.V4) bool {
	return g.value
}

func TestDisableTOTP(t *testing.T) {
	validGuard := disableTOTPGuard{value: true}
	invalidGuard := disableTOTPGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewDisableTOTPHandler(broker, users)

	password := "password"
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))

	if err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}

	tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	_totp := errors.Must(tb.Generate(verifiedTOTPUser.TOTPKey, time.Now()))
	totp := errors.Must(domain.NewTOTP(_totp))

	if err := verifiedTOTPUser.VerifyTOTP(totp, domain.TOTPKindApp); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	// Clean the TOTP from memory so we don't get already used errors in the
	// test for success
	otp.CleanUsedTOTP(_totp)

	t.Run("success with correct TOTP", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithTOTP{
			Email: verifiedTOTPUser.Email.String(),
		})
		wantEvents = append(wantEvents, account.DisabledTOTP{
			Email: verifiedTOTPUser.Email.String(),
		})

		totp := errors.Must(tb.Generate(verifiedTOTPUser.TOTPKey, time.Now()))

		err := handler(ctx, account.DisableTOTP{
			Guard:  validGuard,
			UserID: verifiedTOTPUser.ID.String(),
			TOTP:   totp,
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
			name   string
			guard  account.DisableTOTPGuard
			userID string
			totp   string
			want   error
		}{
			{"unauthorised", invalidGuard, "", "", port.ErrUnauthorised},
			{"incorrect TOTP", validGuard, verifiedTOTPUser.ID.String(), "123456", port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.DisableTOTP{
					Guard:  tc.guard,
					UserID: tc.userID,
					TOTP:   tc.totp,
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
