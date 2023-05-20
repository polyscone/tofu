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

func TestFindUserByEmail(t *testing.T) {
	ctx := context.Background()
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	users := errors.Must(repo.NewSQLiteAccountUserRepo(ctx, db, []byte("s")))
	handler := account.NewFindUserByEmailHandler(broker, users)

	activatedUser := errors.Must(repotest.AddActivatedUser(t, users, ctx, "joe@bloggs.com", "password"))

	t.Run("success with valid user email", func(t *testing.T) {
		var wantEvents []event.Event
		var gotEvents []event.Event
		broker.Clear()
		broker.ListenAny(func(evt event.Event) { gotEvents = append(gotEvents, evt) })

		found, err := handler(ctx, account.FindUserByEmail{
			Email: activatedUser.Email.String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if want, got := activatedUser.ID.String(), found.ID; want != got {
			t.Errorf("want user id %q; got %q", want, got)
		}
		if want, got := activatedUser.Email.String(), found.Email; want != got {
			t.Errorf("want email %q; got %q", want, got)
		}

		testutil.CheckEvents(t, wantEvents, gotEvents)
	})
}
