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
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

type setupTOTPGuard struct {
	canSetupTOTP bool
}

func (g setupTOTPGuard) CanSetupTOTP(userID uuid.V4) bool {
	return g.canSetupTOTP
}

func TestSetupTOTP(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewSetupTOTPHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))
	verifiedTOTPUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "jane@doe.com", password))

	if _, err := verifiedTOTPUser.SetupTOTP(); err != nil {
		t.Fatal(err)
	}

	tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	_totp := errors.Must(tb.Generate(verifiedTOTPUser.TOTPKey, time.Now()))
	totp := errors.Must(domain.NewTOTP(_totp))

	if err := verifiedTOTPUser.VerifyTOTP(totp); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedTOTPUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success with activated logged in user", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		res, err := handler(ctx, account.SetupTOTP{
			Guard:  setupTOTPGuard{canSetupTOTP: true},
			UserID: activatedUser.ID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(users.FindByID(ctx, activatedUser.ID))
		if want, got := res.Key, user.TOTPKey; !bytes.Equal(want, got) {
			t.Errorf("want %q; got %q", want, got)
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
			guard  account.SetupTOTPGuard
			userID string
			want   error
		}{
			{"unauthorised", setupTOTPGuard{canSetupTOTP: false}, activatedUser.ID.String(), port.ErrUnauthorised},
			{"TOTP already setup and verified", setupTOTPGuard{canSetupTOTP: true}, verifiedTOTPUser.ID.String(), port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				_, err := handler(ctx, account.SetupTOTP{
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
