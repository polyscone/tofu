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
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/domain"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/repotest"
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
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewVerifyTOTPHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	setupAppTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))
	setupSMSTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "baz@qux.com", password))
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "foo@bar.com", password))

	if err := setupAppTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := setupAppTOTPUser.ChangeTOTPTelephone(text.GenerateTelephone()); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, setupAppTOTPUser); err != nil {
		t.Fatal(err)
	}

	if err := setupSMSTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := setupSMSTOTPUser.ChangeTOTPTelephone(text.GenerateTelephone()); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, setupSMSTOTPUser); err != nil {
		t.Fatal(err)
	}

	if err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	verifiedTOTPUser.TOTPVerifiedAt = time.Now()
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success with activated user with app TOTP setup", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(setupAppTOTPUser.TOTPKey, time.Now()))

		// Deliberately set the use SMS flag to ensure it's disabled by the command
		setupAppTOTPUser.TOTPUseSMS = true

		err := handler(ctx, account.VerifyTOTP{
			Guard:  validGuard,
			UserID: setupAppTOTPUser.ID.String(),
			TOTP:   totp,
			UseSMS: false,
		})
		if err != nil {
			t.Fatal(err)
		}

		setupAppTOTPUser = errors.Must(users.FindByID(ctx, setupAppTOTPUser.ID))
		if setupAppTOTPUser.TOTPUseSMS {
			t.Error("want TOTP app users to have a false use SMS flag")
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("success with activated user with SMS TOTP setup", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(setupSMSTOTPUser.TOTPKey, time.Now()))

		err := handler(ctx, account.VerifyTOTP{
			Guard:  validGuard,
			UserID: setupSMSTOTPUser.ID.String(),
			TOTP:   totp,
			UseSMS: true,
		})
		if err != nil {
			t.Fatal(err)
		}

		setupSMSTOTPUser = errors.Must(users.FindByID(ctx, setupSMSTOTPUser.ID))
		if !setupSMSTOTPUser.TOTPUseSMS {
			t.Error("want TOTP SMS users to have a true use SMS flag")
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
			{"empty user id correct TOTP", validGuard, "", setupAppTOTPUser.TOTPKey, port.ErrMalformedInput},
			{"empty user id incorrect TOTP", validGuard, "", nil, port.ErrMalformedInput},
			{"no TOTP user id correct TOTP", validGuard, activatedUser.ID.String(), setupAppTOTPUser.TOTPKey, nil},
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

				return !errors.Is(err, port.ErrMalformedInput)
			})
		})

		t.Run("invalid totp input", func(t *testing.T) {
			quick.Check(t, func(totp quick.Invalid[domain.TOTP]) bool {
				err := execute(totp.Unwrap())

				return errors.Is(err, port.ErrMalformedInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
