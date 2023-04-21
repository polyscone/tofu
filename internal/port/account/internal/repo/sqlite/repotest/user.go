package repotest

import (
	"context"
	"reflect"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

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

	// Setting up a TOTP should also save a non-zero number of recovery code
	// strings which should not be empty
	if _, err := found.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, found); err != nil {
		t.Fatalf("want <nil>; got error %q", err)
	}
	found = errors.Must(users.FindByID(ctx, found.ID))
	if len(found.RecoveryCodes) == 0 {
		t.Fatal("want at least one recovery code")
	} else {
		for _, code := range found.RecoveryCodes {
			if len(code) == 0 {
				t.Error("want code; got empty string")
			}
		}
	}

	// Running TOTP setup a second time should replace old codes
	nOldCodes := len(found.RecoveryCodes)
	if _, err := found.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, found); err != nil {
		t.Fatalf("want <nil>; got error %q", err)
	}
	found = errors.Must(users.FindByID(ctx, found.ID))
	if want, got := nOldCodes, len(found.RecoveryCodes); want != got {
		t.Errorf("want %v codes; got %v", want, got)
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

func AddActivatedUser(t *testing.T, users account.UserRepo, ctx context.Context, _email, _password string) (domain.User, error) {
	user, err := AddUser(t, users, ctx, _email)
	if err != nil {
		return domain.User{}, err
	}

	password, err := domain.NewPassword(_password)
	if err != nil {
		return domain.User{}, err
	}

	if err := user.ActivateAndSetPassword(password); err != nil {
		return domain.User{}, err
	}
	if err := users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}

	return users.FindByID(ctx, user.ID)
}
