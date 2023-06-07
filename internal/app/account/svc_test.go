package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/repo/sqlite"
)

var hasher = testutil.NewPasswordHasher()

func NewTestEnv(ctx context.Context) (*account.Service, event.Broker, account.ReadWriter) {
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errors.Must(sqlite.NewAccountRepo(ctx, db))
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
	ActivateTOTP       bool
}

func MustAddUser(t *testing.T, ctx context.Context, repo account.ReadWriter, tu TestUser) *account.User {
	t.Helper()

	tu.VerifyTOTP = tu.VerifyTOTP || tu.ActivateTOTP
	tu.SetupTOTP = tu.SetupTOTP || tu.SetupTOTPTelephone || tu.VerifyTOTP || tu.ActivateTOTP
	tu.Activate = tu.Activate || tu.SetupTOTP || tu.VerifyTOTP

	user := account.NewUser(errors.Must(text.NewEmail(tu.Email)))

	errors.Must0(user.SignUp())

	if tu.Activate {
		if tu.Password == "" {
			tu.Password = "password"
		}

		password := errors.Must(account.NewPassword(tu.Password))

		errors.Must0(user.Activate(password, hasher))
	}
	if tu.SetupTOTP {
		errors.Must0(user.SetupTOTP())
	}
	if tu.SetupTOTPTelephone {
		errors.Must0(user.ChangeTOTPTelephone(text.GenerateTel()))
	}
	if tu.VerifyTOTP {
		alg := errors.Must(otp.NewAlgorithm(user.TOTPAlgorithm))
		tb := errors.Must(otp.NewTimeBased(user.TOTPDigits, alg, time.Unix(0, 0), user.TOTPPeriod))
		totp := errors.Must(account.NewTOTP(errors.Must(tb.Generate(user.TOTPKey, time.Now()))))

		errors.Must0(user.VerifyTOTP(totp, "app"))

		otp.CleanUsedTOTP(totp.String())
	}
	if tu.ActivateTOTP {
		errors.Must0(user.ActivateTOTP())
	}

	errors.Must0(repo.AddUser(ctx, user))

	return user
}

type TestRole struct {
	Name        string
	Permissions []string
}

func MustAddRole(t *testing.T, ctx context.Context, repo account.ReadWriter, tr TestRole) *account.Role {
	t.Helper()

	name := errors.Must(account.NewRoleName(tr.Name))

	var permissions []account.Permission
	if tr.Permissions != nil {
		permissions = make([]account.Permission, len(tr.Permissions))

		for i, name := range tr.Permissions {
			permissions[i] = errors.Must(account.NewPermission(name))
		}
	}

	role := account.NewRole(name, "", permissions)

	errors.Must0(repo.AddRole(ctx, role))

	return role
}
