package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/repo/repotest"
)

func TestFindUserByID(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(account.NewSQLiteUserRepo(ctx, db, []byte("s")))
	authenticateWithPasswordHandler := account.NewAuthenticateWithPasswordHandler(broker, users)
	handler := account.NewFindUserByIDHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))

	authRes := errors.Must(authenticateWithPasswordHandler(ctx, account.AuthenticateWithPassword{
		Email:    activatedUser.Email.String(),
		Password: password,
	}))

	t.Run("success with valid user id", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		user, err := handler(ctx, account.FindUserByID{
			UserID: authRes.UserID,
		})
		if err != nil {
			t.Fatal(err)
		}

		var containsUserID bool
		for _, claim := range user.Claims {
			containsUserID = claim == authRes.UserID
			if containsUserID {
				break
			}
		}
		if want, got := true, containsUserID; want != got {
			t.Error("want claims to contain user id")
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
