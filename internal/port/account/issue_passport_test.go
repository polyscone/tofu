package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
	"github.com/polyscone/tofu/internal/port/account/internal/repo/sqlite/repotest"
)

func TestIssuePassport(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewIssuePassportHandler(broker, users)

	// Seed the repo
	activatedUser := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))
	unactivatedUser := errors.Must(repotest.AddUser(t, users, ctx, "jane@doe.com"))

	password := errors.Must(domain.NewPassword("password"))
	if err := activatedUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, activatedUser); err != nil {
		t.Fatal(err)
	}

	t.Run("success cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name          string
			userID        string
			isAwaitingMFA bool
			isLoggedIn    bool
		}{
			{"activated user id", activatedUser.ID.String(), false, false},
			{"activated user id awaiting MFA", activatedUser.ID.String(), true, false},
			{"activated user id logged in", activatedUser.ID.String(), false, true},
			{"unactivated user id", unactivatedUser.ID.String(), false, false},
			{"unactivated user id awaiting MFA", unactivatedUser.ID.String(), true, false},
			{"unactivated user id logged in", unactivatedUser.ID.String(), false, true},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				passport, err := handler(ctx, account.IssuePassport{
					UserID:        tc.userID,
					IsAwaitingMFA: tc.isAwaitingMFA,
					IsLoggedIn:    tc.isLoggedIn,
				})
				if err != nil {
					t.Fatalf("want <nil>; got %q", err)
				}

				if want, got := tc.userID, passport.UserID(); want != got {
					t.Errorf("want user id %q; got %q", want, got)
				}
				if want, got := tc.isAwaitingMFA, passport.IsAwaitingMFA(); want != got {
					t.Errorf("want awaiting MFA %v; got %v", want, got)
				}
				if want, got := tc.isLoggedIn, passport.IsLoggedIn(); want != got {
					t.Errorf("want logged in %v; got %v", want, got)
				}
			})
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})

	t.Run("error cases", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		tt := []struct {
			name          string
			userID        string
			isAwaitingMFA bool
			isLoggedIn    bool
		}{
			{"empty user id", "", false, false},
			{"nil user id", uuid.Nil.String(), false, false},
			{"activated user id conflicting states", activatedUser.ID.String(), true, true},
			{"unactivated user id conflicting states", unactivatedUser.ID.String(), true, true},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				_, err := handler(ctx, account.IssuePassport{
					UserID:        tc.userID,
					IsAwaitingMFA: tc.isAwaitingMFA,
					IsLoggedIn:    tc.isLoggedIn,
				})
				if err == nil {
					t.Error("want error; got <nil>")
				}
			})
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
