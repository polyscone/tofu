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
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

func TestSetupTOTP(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	authenticateWithPasswordHandler := account.NewAuthenticateWithPasswordHandler(broker, users)
	authenticateWithTOTPHandler := account.NewAuthenticateWithTOTPHandler(broker, users)
	handler := account.NewSetupTOTPHandler(broker, users)

	// Seed the repo
	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))

	if err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	verifiedTOTPUser.VerifyTOTP()
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	activatedUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    activatedUser.Email.String(),
		Password: password,
	}))
	verifiedTOTPUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    verifiedTOTPUser.Email.String(),
		Password: password,
	}))

	tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	totp := errors.Must(tb.Generate(verifiedTOTPUser.TOTPKey, time.Now()))

	verifiedTOTPUserPassport = errors.Must(authenticateWithTOTPHandler(ctx, account.AuthenticateWithTOTP{
		Passport: verifiedTOTPUserPassport,
		TOTP:     totp,
	}))

	t.Run("success with activated logged in user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		err := handler(ctx, account.SetupTOTP{
			Guard:  activatedUserPassport,
			UserID: activatedUserPassport.UserID(),
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
			name     string
			passport account.Passport
			want     error
		}{
			{"empty passport", account.EmptyPassport, port.ErrUnauthorised},
			{"TOTP already setup and verified", verifiedTOTPUserPassport, port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				err := handler(ctx, account.SetupTOTP{
					Guard:  tc.passport,
					UserID: tc.passport.UserID(),
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
