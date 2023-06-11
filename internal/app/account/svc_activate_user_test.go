package account_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
)

func TestActivateUser(t *testing.T) {
	t.Run("success with existing user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user1 := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com"})
		user2 := MustAddUser(t, ctx, store, TestUser{Email: "foo@bar.com"})
		superRole := errsx.Must(store.FindRoleByName(ctx, account.SuperRole.Name))

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		superUserCount := errsx.Must(store.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 0, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}

		if err := svc.ActivateUser(ctx, user1.Email, "password", "password"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.RolesChanged{Email: user1.Email})
		events.Expect(account.Activated{Email: user1.Email})

		user1 = errsx.Must(store.FindUserByEmail(ctx, user1.Email))

		if user1.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}

		if err := svc.SignInWithPassword(ctx, user1.Email, "password"); err != nil {
			t.Errorf("want to be able to sign in with chosen password; got error: %v", err)
		}

		events.Expect(account.SignedInWithPassword{Email: user1.Email})

		superUserCount = errsx.Must(store.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 1, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}

		if err := svc.ActivateUser(ctx, user2.Email, "password", "password"); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.Activated{Email: user2.Email})

		superUserCount = errsx.Must(store.CountUsersByRoleID(ctx, superRole.ID))
		if want, got := 1, superUserCount; want != got {
			t.Fatalf("want super user count to be %v; got %v", want, got)
		}
	})

	t.Run("fail activating an already activated user", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, store := NewTestEnv(ctx)

		user := MustAddUser(t, ctx, store, TestUser{Email: "joe@bloggs.com", Activate: true})

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		if err := svc.ActivateUser(ctx, user.Email, "password", "password"); err == nil {
			t.Error("want error; got <nil>")
		}
	})

	t.Run("properties", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnv(ctx)

		events := testutil.NewEventLog(broker)
		defer events.Check(t)

		execute := func(email account.Email, password, passwordCheck account.Password) error {
			err := svc.ActivateUser(ctx, email.String(), password.String(), passwordCheck.String())
			if err == nil {
				events.Expect(account.Activated{Email: email.String()})
			}

			return err
		}

		t.Run("valid inputs", func(t *testing.T) {
			quick.Check(t, func(email account.Email, password account.Password) bool {
				err := execute(email, password, password)

				return !errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid email", func(t *testing.T) {
			quick.Check(t, func(email quick.Invalid[account.Email], password account.Password) bool {
				err := execute(email.Unwrap(), password, password)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("invalid password", func(t *testing.T) {
			quick.Check(t, func(email account.Email, password quick.Invalid[account.Password]) bool {
				err := execute(email, password.Unwrap(), password.Unwrap())

				return errors.Is(err, app.ErrMalformedInput)
			})
		})

		t.Run("mismatched password", func(t *testing.T) {
			quick.Check(t, func(email account.Email, password account.Password) bool {
				mismatch := bytes.Clone(password)
				mismatch[0]++

				err := execute(email, password, mismatch)

				return errors.Is(err, app.ErrMalformedInput)
			})
		})
	})
}
