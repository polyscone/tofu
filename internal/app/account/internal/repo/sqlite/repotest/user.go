package repotest

import (
	"context"
	"reflect"
	"testing"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/account/internal/domain"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

var testUserRoles = []domain.Role{
	domain.NewRole("Foo", "pages.create", "pages.update"),
	domain.NewRole("Bar", "users.create", "users.delete"),
	domain.NewRole("Baz", "plugins.create"),
	domain.NewRole("Qux"),
}

func RunUserTests(t *testing.T, users account.UserRepo) {
	ctx := context.Background()

	// Seed the repo
	existing := errors.Must(AddUser(t, users, ctx, "joe@bloggs.com"))
	errors.Must(AddUser(t, users, ctx, "jane@doe.com"))

	// An existing user should be found an be populated with correct values
	found := errors.Must(users.FindByEmail(ctx, "joe@bloggs.com"))
	if !reflect.DeepEqual(existing, found) {
		t.Errorf("\nwant %#v\ngot  %#v", existing, found)
	}

	// A non-existent user should not be found
	_, err := users.FindByEmail(ctx, "x@y.z")
	if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
		t.Errorf("want %q; got %q", want, got)
	}

	// Adding a user with a duplicate email should conflict
	_, err = AddUser(t, users, ctx, "jane@doe.com")
	if want, got := repo.ErrConflict, err; !errors.Is(got, want) {
		t.Errorf("want %q; got %q", want, got)
	}

	// Adding a user with a duplicate email with different casing should conflict
	_, err = AddUser(t, users, ctx, "JANE@DOE.COM")
	if want, got := repo.ErrConflict, err; !errors.Is(got, want) {
		t.Errorf("want %q; got %q", want, got)
	}

	// Activating should save a new flag to the database
	password := errors.Must(domain.NewPassword("password"))
	if err := found.ActivateAndSetPassword(password); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, found); err != nil {
		t.Fatalf("want <nil>; got error %q", err)
	}
	found = errors.Must(users.FindByID(ctx, found.ID))
	if found.ActivatedAt.IsZero() {
		t.Error("want non-zero time")
	}

}

func AddUser(t *testing.T, users account.UserRepo, ctx context.Context, _email string) (domain.User, error) {
	t.Helper()

	id := errors.Must(uuid.NewV4())
	email := errors.Must(text.NewEmail(_email))

	user := domain.NewUser(id)

	user.Register(email)

	if err := users.Add(ctx, user); err != nil {
		return domain.User{}, err
	}

	return users.FindByID(ctx, user.ID)
}
