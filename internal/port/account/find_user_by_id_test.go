package account_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/repo"
	"github.com/polyscone/tofu/internal/repo/repotest"
)

func TestFindUserByID(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewFindUserByIDHandler(broker, users)

	password := "password"
	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", password))

	t.Run("success with valid user id", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		user, err := handler(ctx, account.FindUserByID{
			UserID: activatedUser.ID.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		var containsUserID bool
		for _, claim := range user.Claims {
			containsUserID = claim == activatedUser.ID.String()
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
