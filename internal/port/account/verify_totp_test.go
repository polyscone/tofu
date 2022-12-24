package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

type verifyTOTPGuard struct {
	value bool
}

func (g verifyTOTPGuard) CanVerifyTOTP(userID uuid.V4) bool {
	return g.value
}

func TestVerifyTOTP(t *testing.T) {
	validGuard := verifyTOTPGuard{value: true}
	invalidGuard := verifyTOTPGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewVerifyTOTPHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	setupTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "foo@bar.com", password))

	if _, err := setupTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, setupTOTPUser); err != nil {
		t.Fatal(err)
	}

	if _, err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	verifiedTOTPUser.TOTPVerifiedAt = time.Now()
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success with activated user with TOTP setup", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(setupTOTPUser.TOTPKey, time.Now()))

		err := handler(ctx, account.VerifyTOTP{
			Guard:  validGuard,
			UserID: setupTOTPUser.ID.String(),
			TOTP:   totp,
		})
		if err != nil {
			t.Fatal(err)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name    string
			guard   account.VerifyTOTPGuard
			userID  string
			totpKey []byte
			want    error
		}{
			{"unauthorised", invalidGuard, "", nil, port.ErrUnauthorised},
			{"empty user id correct TOTP", validGuard, "", setupTOTPUser.TOTPKey, port.ErrInvalidInput},
			{"empty user id incorrect TOTP", validGuard, "", nil, port.ErrInvalidInput},
			{"no TOTP user id correct TOTP", validGuard, activatedUser.ID.String(), setupTOTPUser.TOTPKey, port.ErrBadRequest},
			{"already verified TOTP", validGuard, verifiedTOTPUser.ID.String(), verifiedTOTPUser.TOTPKey, port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpKey != nil {
					tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
					totp = errors.Must(tb.Generate(tc.totpKey, time.Now()))
				}

				err := handler(ctx, account.VerifyTOTP{
					Guard:  tc.guard,
					UserID: tc.userID,
					TOTP:   totp,
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

		execute := func(totp domain.TOTP) error {
			err := handler(ctx, account.VerifyTOTP{
				Guard:  validGuard,
				UserID: activatedUser.ID.String(),
				TOTP:   totp.String(),
			})

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(totp domain.TOTP) bool {
				err := execute(totp)

				return !errors.Is(err, port.ErrInvalidInput)
			})
		})

		t.Run("invalid totp input", func(t *testing.T) {
			quick.Check(t, func(totp quick.Invalid[domain.TOTP]) bool {
				err := execute(totp.Unwrap())

				return errors.Is(err, port.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
