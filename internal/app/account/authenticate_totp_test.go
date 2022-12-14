package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/app/account/internal/repo/sqlite/repotest"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
)

func TestAuthenticateWithTOTP(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	issuePassportHandler := account.NewIssuePassportHandler(broker, users)
	authenticateWithPasswordHandler := account.NewAuthenticateWithPasswordHandler(broker, users)
	handler := account.NewAuthenticateWithTOTPHandler(broker, users)

	// Seed the repo
	verifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))
	verifiedNoTOTPUser := errors.Must(repotest.AddUser(t, users, ctx, "jim@bloggs.com"))
	unverifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "jane@doe.com"))
	unverifiedTOTPUser := errors.Must(repotest.AddUser(t, users, ctx, "foo@bar.com"))

	password := errors.Must(domain.NewPassword("password"))
	if err := verifiedUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	totpKey := domain.NewTOTPKey([]byte("12345678901234567890"))
	verifiedUser.ChangeTOTPKey(totpKey)
	if err := verifiedUser.VerifyTOTPKey(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedUser); err != nil {
		t.Fatal(err)
	}

	if err := verifiedNoTOTPUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedNoTOTPUser); err != nil {
		t.Fatal(err)
	}

	totpKey = domain.NewTOTPKey([]byte("12345678901234567891"))
	unverifiedUser.ChangeTOTPKey(totpKey)
	if err := users.Save(ctx, unverifiedUser); err != nil {
		t.Fatal(err)
	}

	if err := unverifiedTOTPUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	totpKey = domain.NewTOTPKey([]byte("12345678901234567892"))
	unverifiedTOTPUser.ChangeTOTPKey(totpKey)
	if err := users.Save(ctx, unverifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	verifiedUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    verifiedUser.Email.String(),
		Password: password.String(),
	}))
	verifiedNoTOTPUserPassport := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    verifiedNoTOTPUser.Email.String(),
		Password: password.String(),
	}))
	unverifiedUserPassport := errors.Must(issuePassportHandler(ctx, account.IssuePassport{
		UserID:        unverifiedUser.ID.String(),
		IsAwaitingMFA: true,
	}))
	unverifiedTOTPUserPassport := errors.Must(issuePassportHandler(ctx, account.IssuePassport{
		UserID:        unverifiedTOTPUser.ID.String(),
		IsAwaitingMFA: true,
	}))

	t.Run("success for valid passport from password authentication and correct totp", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.AuthenticatedWithTOTP{
			Email: verifiedUser.Email.String(),
		})

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(verifiedUser.TOTPKey, time.Now()))

		_, err := handler(ctx, account.AuthenticateWithTOTP{
			Passport: verifiedUserPassport,
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
			{"empty passport correct TOTP", account.EmptyPassport, verifiedUser.TOTPKey, app.ErrBadRequest},
			{"empty passport incorrect TOTP", account.EmptyPassport, nil, app.ErrBadRequest},
			{"verified user passport incorrect TOTP", verifiedUserPassport, nil, app.ErrBadRequest},
			{"verified user passport unverified correct TOTP", unverifiedTOTPUserPassport, unverifiedTOTPUser.TOTPKey, app.ErrBadRequest},
			{"verified user passport without TOTP setup", verifiedNoTOTPUserPassport, nil, app.ErrBadRequest},
			{"unverified user passport incorrect TOTP", unverifiedUserPassport, nil, app.ErrBadRequest},
			{"unverified user passport correct TOTP", unverifiedUserPassport, unverifiedUser.TOTPKey, app.ErrBadRequest},
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
				Passport: verifiedUserPassport,
				TOTP:     totp.String(),
			})

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(totp domain.TOTP) bool {
				err := execute(totp)

				return !errors.Is(err, app.ErrInvalidInput)
			})
		})

		t.Run("invalid totp input", func(t *testing.T) {
			quick.Check(t, func(totp quick.Invalid[domain.TOTP]) bool {
				err := execute(totp.Unwrap())

				return errors.Is(err, app.ErrInvalidInput)
			})
		})

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
