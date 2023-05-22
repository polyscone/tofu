package account

import (
	"fmt"
	"time"

	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

const (
	TOTPKindApp string = "app"
	TOTPKindSMS string = "sms"
)

var ErrNotActivated = errors.New("account is not activated")

type User struct {
	aggregate.Root

	ID             uuid.V4
	Email          text.Email
	HashedPassword []byte
	TOTPUseSMS     bool
	TOTPTelephone  text.Telephone
	TOTPKey        TOTPKey
	TOTPAlgorithm  string
	TOTPDigits     int
	TOTPPeriod     time.Duration
	TOTPVerifiedAt time.Time
	RecoveryCodes  []RecoveryCode
	Roles          []Role
	Claims         []Claim
	ActivatedAt    time.Time
}

func NewUser(id uuid.V4) User {
	return User{ID: id}
}

func (u *User) HasVerifiedTOTP() bool {
	return !u.TOTPVerifiedAt.IsZero() && len(u.TOTPKey) != 0
}

func (u *User) Register(email text.Email, password Password, hasher password.Hasher) error {
	u.Email = email

	if err := u.setPassword(password, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(Registered{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) Activate() error {
	if !u.ActivatedAt.IsZero() {
		return errors.Tracef("already activated")
	}

	u.ActivatedAt = time.Now()

	u.Events.Enqueue(Activated{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) setPassword(newPassword Password, hasher password.Hasher) error {
	hashedPassword, err := hasher.EncodedHash(newPassword)
	if err != nil {
		return errors.Tracef(err)
	}

	u.HashedPassword = hashedPassword

	return nil
}

func (u *User) ChangePassword(oldPassword, newPassword Password, hasher password.Hasher) error {
	if u.ActivatedAt.IsZero() {
		return errors.Tracef("cannot change password until activated")
	}

	if _, err := u.verifyPassword(oldPassword, hasher); err != nil {
		errs := errors.Map{"old password": err}

		return errs.Tracef(port.ErrInvalidInput)
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(PasswordChanged{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) ResetPassword(newPassword Password, hasher password.Hasher) error {
	if u.ActivatedAt.IsZero() {
		return errors.Tracef("cannot change password until activated")
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(PasswordReset{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) SetupTOTP() error {
	if u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "TOTP already setup and verified")
	}

	if len(u.TOTPKey) == 0 {
		key, err := NewTOTPKey(otp.SHA1)
		if err != nil {
			return errors.Tracef(err)
		}

		u.TOTPKey = key
		u.TOTPAlgorithm = "SHA1"
		u.TOTPDigits = 6
		u.TOTPPeriod = 30 * time.Second
	}

	if len(u.RecoveryCodes) == 0 {
		nCodes := 6
		u.RecoveryCodes = make([]RecoveryCode, nCodes)

		for i := 0; i < nCodes; i++ {
			code, err := GenerateRecoveryCode()
			if err != nil {
				return errors.Tracef(err)
			}

			u.RecoveryCodes[i] = code
		}
	}

	return nil
}

func (u *User) VerifyTOTP(totp TOTP, kind string) error {
	if kind != TOTPKindApp && kind != TOTPKindSMS {
		panic(fmt.Sprintf("invalid TOTP verification kind %q", kind))
	}

	if u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "already verified")
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, otp.SHA1, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return errors.Tracef(err)
	}

	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
		if errors.Is(err, otp.ErrPasswordUsed) {
			errs := errors.Map{"totp": errors.New("passcode already used")}

			return errs.Tracef(port.ErrInvalidInput)
		}

		return errors.Tracef(err)
	}
	if !ok {
		errs := errors.Map{"totp": errors.New("invalid passcode")}

		return errs.Tracef(port.ErrInvalidInput)
	}

	u.TOTPUseSMS = kind == "sms"
	u.TOTPVerifiedAt = time.Now()

	return nil
}

func (u *User) ChangeTOTPTelephone(newTelephone text.Telephone) error {
	if len(u.TOTPKey) == 0 {
		return errors.Tracef(port.ErrBadRequest, "cannot change TOTP telephone without a key setup")
	}

	if u.TOTPTelephone == newTelephone {
		return nil
	}

	oldTelephone := u.TOTPTelephone

	u.TOTPTelephone = newTelephone

	u.Events.Enqueue(TOTPTelephoneChanged{
		Email:        u.Email.String(),
		OldTelephone: oldTelephone.String(),
		NewTelephone: u.TOTPTelephone.String(),
	})

	return nil
}

func (u *User) GenerateTOTP() (string, error) {
	if len(u.TOTPKey) == 0 {
		return "", errors.Tracef(port.ErrBadRequest, "cannot generate a TOTP without a key setup")
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, otp.SHA1, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return "", errors.Tracef(err)
	}

	totp, err := tb.Generate(u.TOTPKey, time.Now())

	return totp, errors.Tracef(err)
}

func (u *User) RegenerateRecoveryCodes() error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "cannot regenerate recovery codes without a verified TOTP")
	}

	nCodes := 6
	u.RecoveryCodes = make([]RecoveryCode, nCodes)

	for i := 0; i < nCodes; i++ {
		code, err := GenerateRecoveryCode()
		if err != nil {
			return errors.Tracef(err)
		}

		u.RecoveryCodes[i] = code
	}

	u.Events.Enqueue(RecoveryCodesRegenerated{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) DisableTOTP(totp TOTP) error {
	if err := u.AuthenticateWithTOTP(totp); err != nil {
		return errors.Tracef(err)
	}

	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.RecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) DisableTOTPWithRecoveryCode(recoveryCode RecoveryCode) error {
	if err := u.AuthenticateWithRecoveryCode(recoveryCode); err != nil {
		return errors.Tracef(err)
	}

	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.RecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) verifyPassword(password Password, hasher password.Hasher) (bool, error) {
	ok, rehash, err := hasher.Verify(password, u.HashedPassword)
	if err != nil {
		return false, errors.Tracef(err)
	}
	if !ok {
		return false, errors.Tracef(port.ErrBadRequest, "could not verify password")
	}
	if rehash {
		err := u.setPassword(password, hasher)

		return true, errors.Tracef(err)
	}

	return false, nil
}

func (u *User) AuthenticateWithPassword(password Password, hasher password.Hasher) (bool, error) {
	if u.ActivatedAt.IsZero() {
		return false, errors.Tracef(port.ErrBadRequest, ErrNotActivated)
	}

	rehashed, err := u.verifyPassword(password, hasher)
	if err != nil {
		return false, errors.Tracef(err)
	}

	u.Events.Enqueue(AuthenticatedWithPassword{
		Email:          u.Email.String(),
		IsAwaitingTOTP: u.HasVerifiedTOTP(),
	})

	return rehashed, nil
}

func (u *User) AuthenticateWithTOTP(totp TOTP) error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "account does not have TOTP")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(port.ErrBadRequest, ErrNotActivated)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, otp.SHA1, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return errors.Tracef(err)
	}

	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
		if errors.Is(err, otp.ErrPasswordUsed) {
			errs := errors.Map{"totp": errors.New("passcode already used")}

			return errs.Tracef(port.ErrInvalidInput)
		}

		return errors.Tracef(err)
	}
	if !ok {
		errs := errors.Map{"totp": errors.New("invalid passcode")}

		return errs.Tracef(port.ErrInvalidInput)
	}

	u.Events.Enqueue(AuthenticatedWithTOTP{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) AuthenticateWithRecoveryCode(recoveryCode RecoveryCode) error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "account cannot use recovery codes")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(port.ErrBadRequest, ErrNotActivated)
	}

	for i, code := range u.RecoveryCodes {
		if code == recoveryCode {
			u.RecoveryCodes = append(u.RecoveryCodes[:i], u.RecoveryCodes[i+1:]...)

			u.Events.Enqueue(AuthenticatedWithRecoveryCode{
				Email: u.Email.String(),
			})

			return nil
		}
	}

	errs := errors.Map{"recovery code": errors.New("invalid recovery code")}

	return errs.Tracef(port.ErrInvalidInput)
}

func (u *User) Is(query Claim) bool {
	for _, claim := range u.Claims {
		if claim == query {
			return true
		}
	}

	return false
}

func (u *User) Can(query Permission) bool {
	for _, role := range u.Roles {
		for _, permission := range role.Permissions {
			if permission == query {
				return true
			}
		}
	}

	return false
}
