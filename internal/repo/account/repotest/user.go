package repotest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account"
)

func RunUserTests(t *testing.T, users account.UserRepo) {
	ctx := context.Background()

	// Seed the repo
	user1 := errors.Must(AddUser(t, users, ctx, "joe@bloggs.com", "password"))
	errors.Must(AddUser(t, users, ctx, "jane@doe.com", "password"))

	// An existing user should be found an be populated with correct values
	user2 := errors.Must(users.FindByEmail(ctx, "joe@bloggs.com"))
	if !reflect.DeepEqual(user1, user2) {
		t.Errorf("\nwant %#v\ngot  %#v", user1, user2)
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

	// Each populated field should be saved and retrieved
	user2.TOTPUseSMS = true
	user2.TOTPTelephone = "123"
	user2.TOTPKey = errors.Must(account.NewTOTPKey(otp.SHA1))
	user2.TOTPAlgorithm = "SHA1"
	user2.TOTPDigits = 6
	user2.TOTPPeriod = time.Hour
	user2.TOTPVerifiedAt = time.Now().UTC()
	user2.RecoveryCodes = []account.RecoveryCode{errors.Must(account.GenerateRecoveryCode())}
	user2.ActivatedAt = time.Now().UTC()

	if err := users.Save(ctx, user2); err != nil {
		t.Fatalf("want <nil>; got error %q", err)
	}

	user3 := errors.Must(users.FindByID(ctx, user2.ID))
	if !reflect.DeepEqual(user2, user3) {
		t.Errorf("\nwant %#v\ngot  %#v", user2, user3)
	}
}

func AddUser(t *testing.T, users account.UserRepo, ctx context.Context, _email, _password string) (account.User, error) {
	t.Helper()

	id := errors.Must(uuid.NewV4())
	email := errors.Must(text.NewEmail(_email))

	user := account.NewUser(id)

	hasher := testutil.NewPasswordHasher()
	if err := user.Register(email, errors.Must(account.NewPassword(_password)), hasher); err != nil {
		return account.User{}, err
	}

	if err := users.Add(ctx, user); err != nil {
		return account.User{}, err
	}

	return users.FindByID(ctx, user.ID)
}

func AddActivatedUser(t *testing.T, users account.UserRepo, ctx context.Context, _email, _password string) (account.User, error) {
	user, err := AddUser(t, users, ctx, _email, _password)
	if err != nil {
		return account.User{}, err
	}

	if err := user.Activate(); err != nil {
		return account.User{}, err
	}
	if err := users.Save(ctx, user); err != nil {
		return account.User{}, err
	}

	return users.FindByID(ctx, user.ID)
}
