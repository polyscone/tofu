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
	"github.com/polyscone/tofu/internal/port/account/domain"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/repotest"
)

func TestAuthenticateWithTOTP(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewAuthenticateWithTOTPHandler(broker, users)

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

	t.Run("success for valid user id and correct totp", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithTOTP{
			Email: activatedUser.Email.String(),
		})

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(activatedUser.TOTPKey, time.Now()))

		otp.CleanUsedTOTP(totp)

		err := handler(ctx, account.AuthenticateWithTOTP{
			UserID: activatedUser.ID.String(),
			TOTP:   totp,
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
			name    string
			userID  string
			totpKey []byte
			want    error
		}{
			{"empty user id correct TOTP", "", activatedUser.TOTPKey, port.ErrMalformedInput},
			{"empty user id incorrect TOTP", "", nil, port.ErrMalformedInput},
			{"activated user id incorrect TOTP", activatedUser.ID.String(), nil, port.ErrInvalidInput},
			{"activated user id unverified correct TOTP", unverifiedTOTPUser.ID.String(), unverifiedTOTPUser.TOTPKey, port.ErrBadRequest},
			{"activated user id without TOTP setup", activatedNoTOTPUser.ID.String(), nil, port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				totp := "000000"
				if tc.totpKey != nil {
					tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
					totp = errors.Must(tb.Generate(tc.totpKey, time.Now()))
				}

				err := handler(ctx, account.AuthenticateWithTOTP{
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
			err := handler(ctx, account.AuthenticateWithTOTP{
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
