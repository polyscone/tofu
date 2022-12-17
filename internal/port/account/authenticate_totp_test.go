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
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

func TestAuthenticateWithTOTP(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	authenticateWithPasswordHandler := account.NewAuthenticateWithPasswordHandler(broker, users)
	handler := account.NewAuthenticateWithTOTPHandler(broker, users)

	// Seed the repo
	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	activatedNoTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jim@bloggs.com", password))
	unverifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "foo@bar.com", password))

	if _, err := activatedUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	activatedUser.VerifyTOTPKey()
	if err := users.Save(ctx, activatedUser); err != nil {
		t.Fatal(err)
	}

	if _, err := unverifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, unverifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	activatedUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    activatedUser.Email.String(),
		Password: password,
	}))
	activatedNoTOTPUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    activatedNoTOTPUser.Email.String(),
		Password: password,
	}))
	unverifiedTOTPUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    unverifiedTOTPUser.Email.String(),
		Password: password,
	}))

	t.Run("success for valid passport from password authentication and correct totp", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithTOTP{
			Email: activatedUser.Email.String(),
		})

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(activatedUser.TOTPKey, time.Now()))

		_, err := handler(ctx, account.AuthenticateWithTOTP{
			Passport: activatedUserPassport,
			TOTP:     totp,
		})
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name     string
			passport account.Passport
			totpKey  []byte
			want     error
		}{
			{"empty passport correct TOTP", account.EmptyPassport, activatedUser.TOTPKey, port.ErrBadRequest},
			{"empty passport incorrect TOTP", account.EmptyPassport, nil, port.ErrBadRequest},
			{"activated user passport incorrect TOTP", activatedUserPassport, nil, port.ErrBadRequest},
			{"activated user passport unverified correct TOTP", unverifiedTOTPUserPassport, unverifiedTOTPUser.TOTPKey, port.ErrBadRequest},
			{"activated user passport without TOTP setup", activatedNoTOTPUserPassport, nil, port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpKey != nil {
					tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
					totp = errors.Must(tb.Generate(tc.totpKey, time.Now()))
				}

				_, err := handler(ctx, account.AuthenticateWithTOTP{
					Passport: tc.passport,
					TOTP:     totp,
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
			_, err := handler(ctx, account.AuthenticateWithTOTP{
				Passport: activatedUserPassport,
				TOTP:     totp.String(),
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
