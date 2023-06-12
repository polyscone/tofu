package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/testutil"
	"github.com/polyscone/tofu/internal/repository/sqlite"
)

var hasher = testutil.NewPasswordHasher()

func NewTestEnv(ctx context.Context) (*account.Service, event.Broker, account.ReadWriter) {
	broker := event.NewMemoryBroker()
	db := sqlite.OpenInMemoryTestDatabase(ctx)
	repo := errsx.Must(sqlite.NewAccountRepo(ctx, db))
	svc := errsx.Must(account.NewService(broker, repo, hasher))

	return svc, broker, repo
}

type TestUser struct {
	Email        string
	Password     string
	Activate     bool
	SetupTOTP    bool
	SetupTOTPTel bool
	VerifyTOTP   bool
	ActivateTOTP bool
}

func MustAddUser(t *testing.T, ctx context.Context, repo account.ReadWriter, tu TestUser) *account.User {
	t.Helper()

	tu.VerifyTOTP = tu.VerifyTOTP || tu.ActivateTOTP
	tu.SetupTOTP = tu.SetupTOTP || tu.SetupTOTPTel || tu.VerifyTOTP || tu.ActivateTOTP
	tu.Activate = tu.Activate || tu.SetupTOTP || tu.VerifyTOTP

	user := account.NewUser(errsx.Must(account.NewEmail(tu.Email)))

	errsx.Must0(user.SignUp())

	if tu.Activate {
		if tu.Password == "" {
			tu.Password = "password"
		}

		password := errsx.Must(account.NewPassword(tu.Password))

		errsx.Must0(user.Activate(password, hasher))
	}
	if tu.SetupTOTP {
		errsx.Must0(user.SetupTOTP())
	}
	if tu.SetupTOTPTel {
		errsx.Must0(user.ChangeTOTPTel(account.GenerateTel()))
	}
	if tu.VerifyTOTP {
		alg := errsx.Must(otp.NewAlgorithm(user.TOTPAlgorithm))
		tb := errsx.Must(otp.NewTimeBased(user.TOTPDigits, alg, time.Unix(0, 0), user.TOTPPeriod))
		totp := errsx.Must(account.NewTOTP(errsx.Must(tb.Generate(user.TOTPKey, time.Now()))))

		errsx.Must0(user.VerifyTOTP(totp, "app"))

		otp.CleanUsedTOTP(totp.String())
	}
	if tu.ActivateTOTP {
		errsx.Must0(user.ActivateTOTP())
	}

	errsx.Must0(repo.AddUser(ctx, user))

	return user
}

type TestRole struct {
	Name        string
	Permissions []string
}

func MustAddRole(t *testing.T, ctx context.Context, repo account.ReadWriter, tr TestRole) *account.Role {
	t.Helper()

	name := errsx.Must(account.NewRoleName(tr.Name))

	var permissions []account.Permission
	if tr.Permissions != nil {
		permissions = make([]account.Permission, len(tr.Permissions))

		for i, name := range tr.Permissions {
			permissions[i] = errsx.Must(account.NewPermission(name))
		}
	}

	role := account.NewRole(name, "", permissions)

	errsx.Must0(repo.AddRole(ctx, role))

	return role
}
