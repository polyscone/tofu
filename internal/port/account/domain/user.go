package domain

import (
	"time"

	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/password/argon2"
	"github.com/polyscone/tofu/internal/pkg/size"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port"
)

var ErrNotActivated = errors.New("account is not activated")

type Registered struct {
	Email string
}

type Activated struct {
	Email string
}

type AuthenticatedWithPassword struct {
	Email          string
	IsAwaitingTOTP bool
}

type AuthenticatedWithTOTP struct {
	Email string
}

type AuthenticatedWithRecoveryCode struct {
	Email string
}

type DisabledTOTP struct {
	Email string
}

type RecoveryCodesRegenerated struct {
	Email string
}

type PasswordChanged struct {
	Email string
}

type PasswordReset struct {
	Email string
}

type User struct {
	aggregate.Root

	ID             uuid.V4
	Email          text.Email
	HashedPassword []byte
	TOTPKey        TOTPKey
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

func (u *User) Register(email text.Email, password Password) error {
	u.Email = email

	if err := u.setPassword(password); err != nil {
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

func (u *User) VerifyTOTPKey() error {
	if len(u.TOTPKey) == 0 {
		return errors.Tracef("cannot verify an empty TOTP key")
	}

	u.TOTPVerifiedAt = time.Now()

	return nil
}

func (u *User) setPassword(newPassword Password) error {
	const mebibyte = 1 * size.Mebibyte / size.Kibibyte

	// TODO: automatically detect argon2 parameters and use those instead
	hashedPassword, err := argon2.EncodedHash(newPassword, argon2.Params{
		Variant:     argon2.ID,
		Iterations:  1,
		Memory:      64 * mebibyte,
		Parallelism: 4,
		SaltLength:  8,
		KeyLength:   16,
	})
	if err != nil {
		return errors.Tracef(err)
	}

	u.HashedPassword = hashedPassword

	return nil
}

func (u *User) ChangePassword(oldPassword, newPassword Password) error {
	if u.ActivatedAt.IsZero() {
		return errors.Tracef("cannot change password until activated")
	}

	if err := u.verifyPassword(oldPassword); err != nil {
		errs := errors.Map{"old password": err}

		return errs.Tracef(port.ErrInvalidInput)
	}

	if err := u.setPassword(newPassword); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(PasswordChanged{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) ResetPassword(newPassword Password) error {
	if u.ActivatedAt.IsZero() {
		return errors.Tracef("cannot change password until activated")
	}

	if err := u.setPassword(newPassword); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(PasswordReset{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) SetupTOTP() (TOTPParams, error) {
	if u.HasVerifiedTOTP() {
		return TOTPParams{}, errors.Tracef(port.ErrBadRequest, "TOTP already setup and verified")
	}

	key, err := NewTOTPKey(otp.SHA1)
	if err != nil {
		return TOTPParams{}, errors.Tracef(err)
	}

	nCodes := 6

	u.TOTPKey = key
	u.RecoveryCodes = make([]RecoveryCode, nCodes)

	for i := 0; i < nCodes; i++ {
		code, err := GenerateRecoveryCode()
		if err != nil {
			return TOTPParams{}, errors.Tracef(err)
		}

		u.RecoveryCodes[i] = code
	}

	params := TOTPParams{
		Key:       u.TOTPKey,
		Algorithm: "SHA1",
		Digits:    6,
		Period:    30,
	}

	return params, nil
}

func (u *User) VerifyTOTP(totp TOTP) error {
	if u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "already verified")
	}

	tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
		return errors.Tracef(err)
	}
	if !ok {
		errs := errors.Map{"totp": errors.New("invalid passcode")}

		return errs.Tracef(port.ErrInvalidInput)
	}

	u.TOTPVerifiedAt = time.Now()

	return nil
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

	u.TOTPKey = TOTPKey{}
	u.TOTPVerifiedAt = time.Time{}

	u.Events.Enqueue(DisabledTOTP{
		Email: u.Email.String(),
	})

	return nil
}

func (u *User) verifyPassword(password Password) error {
	ok, _, err := argon2.Verify(password, u.HashedPassword, nil)
	if err != nil {
		return errors.Tracef(err)
	}
	if !ok {
		return errors.Tracef(port.ErrBadRequest, "could not verify password")
	}

	return nil
}

func (u *User) AuthenticateWithPassword(password Password) error {
	if u.ActivatedAt.IsZero() {
		return errors.Tracef(port.ErrBadRequest, ErrNotActivated)
	}

	if err := u.verifyPassword(password); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(AuthenticatedWithPassword{
		Email:          u.Email.String(),
		IsAwaitingTOTP: u.HasVerifiedTOTP(),
	})

	return nil
}

func (u *User) AuthenticateWithTOTP(totp TOTP) error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(port.ErrBadRequest, "account does not have TOTP")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(port.ErrBadRequest, ErrNotActivated)
	}

	tb := errors.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
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

	return errors.Tracef(port.ErrInvalidInput, "invalid recovery code")
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
