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
	repo := errsx.Must(sqlite.NewAccountRepo(ctx, db, 1*time.Minute))
	svc := errsx.Must(account.NewService(broker, repo, hasher))

	return svc, broker, repo
}

type TestUser struct {
	Email            string
	Password         string
	Invited          bool
	Verify           bool
	VerifyNoPassword bool
	Activate         bool
	SetupTOTP        bool
	SetupTOTPTel     bool
	VerifyTOTP       bool
	ActivateTOTP     bool
}

func MustAddUserRecoveryCodes(t *testing.T, ctx context.Context, repo account.ReadWriter, tu TestUser) (*account.User, []string) {
	t.Helper()

	tu.VerifyTOTP = tu.VerifyTOTP || tu.ActivateTOTP
	tu.SetupTOTP = tu.SetupTOTP || tu.SetupTOTPTel || tu.VerifyTOTP || tu.ActivateTOTP
	tu.Activate = tu.Activate || tu.SetupTOTP || tu.VerifyTOTP
	tu.Verify = tu.Verify || tu.Activate

	if tu.VerifyNoPassword {
		tu.Password = ""
		tu.Verify = false
		tu.SetupTOTP = false
		tu.SetupTOTPTel = false
		tu.VerifyTOTP = false
		tu.ActivateTOTP = false
	}

	var codes []string
	user := account.NewUser(errsx.Must(account.NewEmail(tu.Email)))

	switch {
	case tu.Invited:
		errsx.Must0(user.InviteUser())

	case tu.VerifyNoPassword:
		errsx.Must0(user.SignUpWithGoogle())

	default:
		errsx.Must0(user.SignUp())
	}

	if tu.Verify {
		if tu.Password == "" {
			tu.Password = "password"
		}

		password := errsx.Must(account.NewPassword(tu.Password))

		errsx.Must0(user.Verify(password, hasher))
	}
	if tu.Activate {
		errsx.Must0(user.Activate())
	}
	if tu.SetupTOTP {
		errsx.Must0(user.SetupTOTP())
	}
	if tu.SetupTOTPTel {
		errsx.Must0(user.ChangeTOTPTel("+00 00 0000 0000"))
	}
	if tu.VerifyTOTP {
		totp := errsx.Must(user.GenerateTOTP())
		codes = errsx.Must(user.VerifyTOTP(errsx.Must(account.NewTOTP(totp)), "app"))

		otp.CleanUsedTOTP(totp)
	}
	if tu.ActivateTOTP {
		errsx.Must0(user.ActivateTOTP())
	}

	errsx.Must0(repo.AddUser(ctx, user))

	return user, codes
}

func MustAddUser(t *testing.T, ctx context.Context, repo account.ReadWriter, tu TestUser) *account.User {
	user, _ := MustAddUserRecoveryCodes(t, ctx, repo, tu)

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
