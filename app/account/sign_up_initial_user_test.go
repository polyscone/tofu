package account_test

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/testx"
)

func TestSignUpInitialUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, repo := NewTestEnvWithSystem(ctx, "site")

		role1 := MustAddRole(t, ctx, repo, TestRole{Name: "Role 1", Permissions: []string{"1", "2"}})
		role2 := MustAddRole(t, ctx, repo, TestRole{Name: "Role 2", Permissions: []string{"2", "3"}})
		roleIDs := []int{role1.ID, role2.ID}

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		password := errsx.Must(account.NewPassword("password123"))
		if _, err := svc.SignUpInitialUser(ctx, "foo@example.com", password.String(), password.String(), roleIDs); err != nil {
			t.Fatal(err)
		}

		events.Expect(account.InitialUserSignedUp{
			Email:  "foo@example.com",
			System: "site",
			Method: account.SignUpMethodSystemSetup,
		})

		user := errsx.Must(repo.FindUserByEmail(ctx, "foo@example.com"))

		if len(user.HashedPassword) == 0 {
			t.Fatal("want user to have a hashed password")
		}
		if user.SignedUpAt.IsZero() {
			t.Error("want signed up at to be populated")
		}
		if want, got := "site", user.SignedUpSystem; want != got {
			t.Errorf("want signed up system to be %q; got %q", want, got)
		}
		if want, got := account.SignUpMethodSystemSetup, user.SignedUpMethod; want != got {
			t.Errorf("want signed up method to be %q; got %q", want, got)
		}
		if user.VerifiedAt.IsZero() {
			t.Error("want non-zero verified at; got zero")
		}
		if user.ActivatedAt.IsZero() {
			t.Error("want non-zero activated at; got zero")
		}
		if !user.SignedUpAt.Equal(user.VerifiedAt) {
			t.Error("want signed up at and verified at to be the same")
		}
		if !user.VerifiedAt.Equal(user.ActivatedAt) {
			t.Error("want verified at and activated at to be the same")
		}

		if want, got := []*account.Role{role1, role2}, user.Roles; len(want) != len(got) {
			t.Errorf("want %v roles; got %v", len(want), len(got))
		} else {
			slices.SortFunc(want, func(a, b *account.Role) int { return cmp.Compare(a.ID, b.ID) })
			slices.SortFunc(got, func(a, b *account.Role) int { return cmp.Compare(a.ID, b.ID) })

			for i, wantRole := range want {
				gotRole := got[i]

				if wantRole.ID != gotRole.ID {
					t.Errorf("want role %q; got %q", wantRole.Name, gotRole.Name)
				}
			}
		}

		if _, err := user.SignInWithPassword("site", password, hasher); err != nil {
			t.Errorf("want to be able to sign in with new password; got %q", err)
		}

		if _, err := svc.SignUpInitialUser(ctx, "bar@example.com", "password", "password", nil); err == nil {
			t.Error("want error after initial user created; got <nil>")
		}
	})

	t.Run("input validation", func(t *testing.T) {
		ctx := context.Background()
		svc, broker, _ := NewTestEnvWithSystem(ctx, "site")

		events := testx.NewEventLog(broker)
		defer events.Check(t)

		tt := []struct {
			name          string
			email         string
			password      string
			passwordCheck string
			isValidInput  bool
		}{
			{"valid inputs", "foo@example.com", "password", "password", true},

			{"invalid email empty", "", "password", "password", false},
			{"invalid email whitespace", "   ", "password", "password", false},
			{"invalid email missing @", "fooexample.com", "password", "password", false},
			{"invalid email part before @", "@example.com", "password", "password", false},
			{"invalid email part after @", "foo@", "password", "password", false},
			{"invalid email includes name", "Foo Bar <foo@example.com>", "password", "password", false},
			{"invalid email missing TLD", "foo@example.", "password", "password", false},

			{"invalid password empty", "foo@example.com", "", "", false},
			{"invalid password whitespace", "foo@example.com", "        ", "        ", false},
			{"invalid password too short", "foo@example.com", ".......", ".......", false},
			{"invalid password too long", "foo@example.com", strings.Repeat(".", 1001), strings.Repeat(".", 1001), false},
			{"invalid password check mismatch", "foo@example.com", "password", "password1", false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.SignUpInitialUser(ctx, tc.email, tc.password, tc.passwordCheck, nil)
				switch {
				case err == nil:
					events.Expect(account.InitialUserSignedUp{
						Email:  tc.email,
						System: "site",
						Method: account.SignUpMethodSystemSetup,
					})

				case tc.isValidInput && errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want any other error value; got %v", app.ErrMalformedInput)

				case !tc.isValidInput && !errors.Is(err, app.ErrMalformedInput):
					t.Errorf("want error: %v; got %v", app.ErrMalformedInput, err)
				}
			})
		}
	})
}
