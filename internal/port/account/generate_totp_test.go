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
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/account/repotest"
)

type generateTOTPGuard struct {
	value bool
}

func (g generateTOTPGuard) CanGenerateTOTP(userID uuid.V4) bool {
	return g.value
}

func TestGenerateTOTP(t *testing.T) {
	validGuard := generateTOTPGuard{value: true}
	invalidGuard := generateTOTPGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewGenerateTOTPHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	setupTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "foo@bar.com", password))

	if err := setupTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, setupTOTPUser); err != nil {
		t.Fatal(err)
	}

	if err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	verifiedTOTPUser.TOTPVerifiedAt = time.Now()
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success with TOTP verified user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
		totp := errors.Must(tb.Generate(verifiedTOTPUser.TOTPKey, time.Now()))

		res, err := handler(ctx, account.GenerateTOTP{
			Guard:  validGuard,
			UserID: verifiedTOTPUser.ID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if want, got := totp, res.TOTP; want != got {
			t.Errorf("want TOTP %q; got %q", want, got)
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
			guard   account.GenerateTOTPGuard
			userID  string
			totpKey []byte
			want    error
		}{
			{"unauthorised", invalidGuard, "", nil, port.ErrUnauthorised},
			{"empty user id correct TOTP", validGuard, "", setupTOTPUser.TOTPKey, port.ErrMalformedInput},
			{"empty user id incorrect TOTP", validGuard, "", nil, port.ErrMalformedInput},
			{"no TOTP user id correct TOTP", validGuard, activatedUser.ID.String(), setupTOTPUser.TOTPKey, nil},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				_, err := handler(ctx, account.GenerateTOTP{
					Guard:  tc.guard,
					UserID: tc.userID,
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
