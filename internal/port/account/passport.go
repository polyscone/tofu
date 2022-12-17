package account

import (
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

var EmptyPassport Passport

type Passport struct {
	user domain.User
}

func newPassport(user domain.User) Passport {
	return Passport{user: user}
}

func (p Passport) UserID() string { return p.user.ID.String() }
func (p Passport) Email() string  { return p.user.Email.String() }

func (p Passport) IsAwaitingTOTP() bool { return p.user.AuthStatus() == domain.AwaitingTOTP }
func (p Passport) IsLoggedIn() bool     { return p.user.AuthStatus() == domain.Authenticated }

func (p Passport) CanChangePassword(userID uuid.V4) bool {
	return p.IsLoggedIn() && p.user.ID == userID
}

func (p Passport) CanSetupTOTP(userID uuid.V4) bool {
	return p.IsLoggedIn() && p.user.ID == userID
}
