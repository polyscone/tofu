package account_test

import (
	"context"
	"sort"
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
	"github.com/polyscone/tofu/internal/port/account/repo/repotest"
)

type regenerateRecoveryCodesGuard struct {
	value bool
}

func (g regenerateRecoveryCodesGuard) CanRegenerateRecoveryCodes(userID uuid.V4) bool {
	return g.value
}

func TestRegenRecoveryCodes(t *testing.T) {
	validGuard := regenerateRecoveryCodesGuard{value: true}
	invalidGuard := regenerateRecoveryCodesGuard{value: false}

	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db, []byte("s")))
	handler := account.NewRegenerateRecoveryCodesHandler(broker, users)

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

	t.Run("success with user with verified TOTP", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		wantEvents = append(wantEvents, account.RecoveryCodesRegenerated{
			Email: verifiedTOTPUser.Email.String(),
		})

		originalRecoveryCodes := verifiedTOTPUser.RecoveryCodes

		res, err := handler(ctx, account.RegenerateRecoveryCodes{
			Guard:  validGuard,
			UserID: verifiedTOTPUser.ID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		user := errors.Must(users.FindByID(ctx, verifiedTOTPUser.ID))

		if len(res.RecoveryCodes) == 0 {
			t.Error("want at least one recovery code; got none")
		} else {
			for _, code := range res.RecoveryCodes {
				if len(code) == 0 {
					t.Fatal("want code; got empty string")
				}
			}
		}

		if want, got := len(res.RecoveryCodes), len(user.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			sort.Strings(res.RecoveryCodes)
			sort.Slice(user.RecoveryCodes, func(i, j int) bool {
				return user.RecoveryCodes[i].String() < user.RecoveryCodes[j].String()
			})

			for i, want := range res.RecoveryCodes {
				if got := user.RecoveryCodes[i]; want != got.String() {
					t.Errorf("want code %q; got %q", want, got)
				}
			}
		}

		if want, got := len(originalRecoveryCodes), len(user.RecoveryCodes); want != got {
			t.Errorf("want %v recovery codes; got %v", want, got)
		} else {
			sort.Slice(originalRecoveryCodes, func(i, j int) bool {
				return originalRecoveryCodes[i].String() < originalRecoveryCodes[j].String()
			})

			sort.Slice(user.RecoveryCodes, func(i, j int) bool {
				return user.RecoveryCodes[i].String() < user.RecoveryCodes[j].String()
			})

			for i, want := range originalRecoveryCodes {
				if got := user.RecoveryCodes[i]; want.String() == got.String() {
					t.Errorf("want different codes; got %q and %q", want, got)
				}
			}
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
			guard  account.RegenerateRecoveryCodesGuard
			userID string
			want   error
		}{
			{"unauthorised", invalidGuard, "", port.ErrUnauthorised},
			{"TOTP not setup or verified", validGuard, activatedUser.ID.String(), port.ErrBadRequest},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				_, err := handler(ctx, account.RegenerateRecoveryCodes{
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
