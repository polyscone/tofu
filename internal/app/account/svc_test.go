package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

var hasher = testutil.NewPasswordHasher()

func NewTestEnv(ctx context.Context) (*account.Service, event.Broker, account.ReadWriter) {
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(sqlite.NewAccountRepo(ctx, db, []byte("secret123")))
	svc := account.NewService(broker, repo, hasher)

	return svc, broker, repo
}

type TestUser struct {
	Email              string
	Password           string
	Activate           bool
	SetupTOTP          bool
	SetupTOTPTelephone bool
	VerifyTOTP         bool
}

func MustAddUser(t *testing.T, ctx context.Context, repo account.ReadWriter, u TestUser) *account.User {
	t.Helper()

	u.SetupTOTP = u.SetupTOTP || u.SetupTOTPTelephone || u.VerifyTOTP
	u.Activate = u.Activate || u.SetupTOTP || u.VerifyTOTP

	user := account.NewUser(errors.Must(text.NewEmail(u.Email)))
	password := errors.Must(account.NewPassword(u.Password))

	errors.Must0(user.Register(password, hasher))

	if u.Activate {
		errors.Must0(user.Activate())
	}
	if u.SetupTOTP {
		errors.Must0(user.SetupTOTP())
	}
	if u.SetupTOTPTelephone {
		errors.Must0(user.ChangeTOTPTelephone(text.GenerateTelephone()))
	}
	if u.VerifyTOTP {
		user.TOTPMethod = account.TOTPMethodApp.String()
		user.TOTPVerifiedAt = time.Now()
	}

	errors.Must0(repo.AddUser(ctx, user))

	return user
}
