package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/app/account/internal/repo/sqlite/repotest"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

func TestIssuePassport(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db))
	handler := account.NewIssuePassportHandler(broker, users)

	// Seed the repo
	verifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "joe@bloggs.com"))
	unverifiedUser := errors.Must(repotest.AddUser(t, users, ctx, "jane@doe.com"))

	password := errors.Must(domain.NewPassword("password"))
	if err := verifiedUser.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, verifiedUser); err != nil {
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
			{"verified user id", verifiedUser.ID.String(), false, false},
			{"verified user id awaiting MFA", verifiedUser.ID.String(), true, false},
			{"verified user id logged in", verifiedUser.ID.String(), false, true},
			{"unverified user id", unverifiedUser.ID.String(), false, false},
			{"unverified user id awaiting MFA", unverifiedUser.ID.String(), true, false},
			{"unverified user id logged in", unverifiedUser.ID.String(), false, true},
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
			{"verified user id conflicting states", verifiedUser.ID.String(), true, true},
			{"unverified user id conflicting states", unverifiedUser.ID.String(), true, true},
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