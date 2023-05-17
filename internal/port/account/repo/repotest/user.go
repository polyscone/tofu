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
	"github.com/polyscone/tofu/internal/port/account/domain"
)

func RunUserTests(t *testing.T, users account.UserRepo) {
	ctx := context.Background()

	// Seed the repo
	existing := errors.Must(AddUser(t, users, ctx, "joe@bloggs.com", "password"))
	errors.Must(AddUser(t, users, ctx, "jane@doe.com", "password"))

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
	_, err = AddUser(t, users, ctx, "jane@doe.com", "password")
	if want, got := repo.ErrConflict, err; !errors.Is(got, want) {
		t.Errorf("want %q; got %q", want, got)
	}

	// Adding a user with a duplicate email with different casing should conflict
	_, err = AddUser(t, users, ctx, "JANE@DOE.COM", "password")
	if want, got := repo.ErrConflict, err; !errors.Is(got, want) {
		t.Errorf("want %q; got %q", want, got)
	}

	// Activating should save a new flag to the database
	if err := found.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, found); err != nil {
		t.Fatalf("want <nil>; got error %q", err)
	}
	found = errors.Must(users.FindByID(ctx, found.ID))
	if found.ActivatedAt.IsZero() {
		t.Error("want non-zero time")
	}

	// Setting up a TOTP should populate the TOTP parameters
	//
	// It should also save a non-zero number of recovery code
	// strings which should not be empty
	if err := found.SetupTOTP(); err != nil {
		t.Fatal(err)
	}
	if err := users.Save(ctx, found); err != nil {
		t.Fatalf("want <nil>; got error %q", err)
	}
	found = errors.Must(users.FindByID(ctx, found.ID))
	if len(found.TOTPKey) == 0 {
		t.Error("want the TOTP key to be populated")
	}
	if len(found.TOTPAlgorithm) == 0 {
		t.Error("want the TOTP algorithm to be populated")
	}
	if found.TOTPDigits == 0 {
		t.Error("want the TOTP digits to be populated")
	}
	if found.TOTPPeriod == 0 {
		t.Error("want the TOTP period to be populated")
	}

	if len(found.RecoveryCodes) == 0 {
		t.Error("want at least one recovery code")
	} else {
		for _, code := range found.RecoveryCodes {
			if len(code) == 0 {
				t.Error("want code; got empty string")
			}
		}
	}
}

func AddUser(t *testing.T, users account.UserRepo, ctx context.Context, _email, _password string) (domain.User, error) {
	t.Helper()

	id := errors.Must(uuid.NewV4())
	email := errors.Must(text.NewEmail(_email))

	user := domain.NewUser(id)

	if err := user.Register(email, errors.Must(domain.NewPassword(_password))); err != nil {
		return domain.User{}, err
	}

	if err := users.Add(ctx, user); err != nil {
		return domain.User{}, err
	}

	return users.FindByID(ctx, user.ID)
}

func AddActivatedUser(t *testing.T, users account.UserRepo, ctx context.Context, _email, _password string) (domain.User, error) {
	user, err := AddUser(t, users, ctx, _email, _password)
	if err != nil {
		return domain.User{}, err
	}

	if err := user.Activate(); err != nil {
		return domain.User{}, err
	}
	if err := users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}

	return users.FindByID(ctx, user.ID)
}
