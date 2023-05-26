package account

import (
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
	"github.com/polyscone/tofu/internal/pkg/password"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

var ErrNotActivated = errors.New("account is not activated")

type User struct {
	aggregate.Root

	ID             int
	Email          string
	HashedPassword []byte
	TOTPMethod     string
	TOTPTelephone  string
	TOTPKey        []byte
	TOTPAlgorithm  string
	TOTPDigits     int
	TOTPPeriod     time.Duration
	TOTPVerifiedAt time.Time
	RegisteredAt   time.Time
	ActivatedAt    time.Time
	LastLoggedInAt time.Time
	Roles          []*Role
	RecoveryCodes  []*RecoveryCode
}

type UserFilter struct {
	ID     *int
	Email  *string
	Search *string

	Limit  int
	Offset int
}

func NewUser(email text.Email) *User {
	return &User{Email: email.String()}
}

func (u *User) HasVerifiedTOTP() bool {
	return !u.TOTPVerifiedAt.IsZero() && len(u.TOTPKey) != 0
}

func (u *User) Register(password Password, hasher password.Hasher) error {
	if err := u.setPassword(password, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.RegisteredAt = time.Now()

	u.Events.Enqueue(Registered{
		Email: u.Email,
	})

	return nil
}

func (u *User) Activate() error {
	if !u.ActivatedAt.IsZero() {
		return errors.Tracef("already activated")
	}

	u.ActivatedAt = time.Now()

	u.Events.Enqueue(Activated{
		Email: u.Email,
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

		return errs.Tracef(app.ErrInvalidInput)
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.Events.Enqueue(PasswordChanged{
		Email: u.Email,
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
		Email: u.Email,
	})

	return nil
}

func (u *User) SetupTOTP() error {
	if u.HasVerifiedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "TOTP already setup and verified")
	}

	if len(u.TOTPKey) == 0 {
		key, err := NewTOTPKey(otp.SHA1)
		if err != nil {
			return errors.Tracef(err)
		}

		u.TOTPKey = key
		u.TOTPAlgorithm = otp.SHA1.String()
		u.TOTPDigits = 6
		u.TOTPPeriod = 30 * time.Second
	}

	if len(u.RecoveryCodes) == 0 {
		nCodes := 6
		u.RecoveryCodes = make([]*RecoveryCode, nCodes)

		for i := 0; i < nCodes; i++ {
			code, err := GenerateCode()
			if err != nil {
				return errors.Tracef(err)
			}

			u.RecoveryCodes[i] = NewRecoveryCode(code)
		}
	}

	return nil
}

func (u *User) VerifyTOTP(totp TOTP, method TOTPMethod) error {
	if u.HasVerifiedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "already verified")
	}

	alg, err := otp.NewAlgorithm(u.TOTPAlgorithm)
	if err != nil {
		return errors.Tracef(err)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, alg, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return errors.Tracef(err)
	}

	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
		if errors.Is(err, otp.ErrPasswordUsed) {
			errs := errors.Map{"totp": errors.New("passcode already used")}

			return errs.Tracef(app.ErrInvalidInput)
		}

		return errors.Tracef(err)
	}
	if !ok {
		errs := errors.Map{"totp": errors.New("invalid passcode")}

		return errs.Tracef(app.ErrInvalidInput)
	}

	u.TOTPMethod = method.String()
	u.TOTPVerifiedAt = time.Now()

	return nil
}

func (u *User) ChangeTOTPTelephone(newTelephone text.Telephone) error {
	if len(u.TOTPKey) == 0 {
		return errors.Tracef(app.ErrBadRequest, "cannot change TOTP telephone without a key setup")
	}

	if u.TOTPTelephone == newTelephone.String() {
		return nil
	}

	oldTelephone := u.TOTPTelephone

	u.TOTPTelephone = newTelephone.String()

	u.Events.Enqueue(TOTPTelephoneChanged{
		Email:        u.Email,
		OldTelephone: oldTelephone,
		NewTelephone: u.TOTPTelephone,
	})

	return nil
}

func (u *User) GenerateTOTP() (string, error) {
	if len(u.TOTPKey) == 0 {
		return "", errors.Tracef(app.ErrBadRequest, "cannot generate a TOTP without a key setup")
	}

	alg, err := otp.NewAlgorithm(u.TOTPAlgorithm)
	if err != nil {
		return "", errors.Tracef(err)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, alg, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return "", errors.Tracef(err)
	}

	totp, err := tb.Generate(u.TOTPKey, time.Now())

	return totp, errors.Tracef(err)
}

func (u *User) RegenerateRecoveryCodes() error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "cannot regenerate recovery codes without a verified TOTP")
	}

	nCodes := 6
	u.RecoveryCodes = make([]*RecoveryCode, nCodes)

	for i := 0; i < nCodes; i++ {
		code, err := GenerateCode()
		if err != nil {
			return errors.Tracef(err)
		}

		u.RecoveryCodes[i] = NewRecoveryCode(code)
	}

	u.Events.Enqueue(RecoveryCodesRegenerated{
		Email: u.Email,
	})

	return nil
}

func (u *User) DisableTOTP(password Password, hasher password.Hasher) error {
	if _, err := u.AuthenticateWithPassword(password, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.TOTPMethod = TOTPMethodNone.String()
	u.TOTPTelephone = ""
	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.RecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{
		Email: u.Email,
	})

	return nil
}

func (u *User) DisableTOTPWithRecoveryCode(code Code) error {
	if err := u.AuthenticateWithRecoveryCode(code); err != nil {
		return errors.Tracef(err)
	}

	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.RecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{
		Email: u.Email,
	})

	return nil
}

func (u *User) verifyPassword(password Password, hasher password.Hasher) (bool, error) {
	ok, rehash, err := hasher.Verify(password, u.HashedPassword)
	if err != nil {
		return false, errors.Tracef(err)
	}
	if !ok {
		return false, errors.Tracef(app.ErrBadRequest, "could not verify password")
	}
	if rehash {
		err := u.setPassword(password, hasher)

		return true, errors.Tracef(err)
	}

	return false, nil
}

func (u *User) AuthenticateWithPassword(password Password, hasher password.Hasher) (bool, error) {
	if u.ActivatedAt.IsZero() {
		return false, errors.Tracef(app.ErrBadRequest, ErrNotActivated)
	}

	rehashed, err := u.verifyPassword(password, hasher)
	if err != nil {
		return false, errors.Tracef(err)
	}

	if !u.HasVerifiedTOTP() {
		u.LastLoggedInAt = time.Now()
	}

	u.Events.Enqueue(AuthenticatedWithPassword{
		Email: u.Email,
	})

	return rehashed, nil
}

func (u *User) AuthenticateWithTOTP(totp TOTP) error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "account does not have TOTP")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(app.ErrBadRequest, ErrNotActivated)
	}

	alg, err := otp.NewAlgorithm(u.TOTPAlgorithm)
	if err != nil {
		return errors.Tracef(err)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, alg, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return errors.Tracef(err)
	}

	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
		if errors.Is(err, otp.ErrPasswordUsed) {
			errs := errors.Map{"totp": errors.New("passcode already used")}

			return errs.Tracef(app.ErrInvalidInput)
		}

		return errors.Tracef(err)
	}
	if !ok {
		errs := errors.Map{"totp": errors.New("invalid passcode")}

		return errs.Tracef(app.ErrInvalidInput)
	}

	u.LastLoggedInAt = time.Now()

	u.Events.Enqueue(AuthenticatedWithTOTP{
		Email: u.Email,
	})

	return nil
}

func (u *User) AuthenticateWithRecoveryCode(code Code) error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "account cannot use recovery codes")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(app.ErrBadRequest, ErrNotActivated)
	}

	for i, rc := range u.RecoveryCodes {
		if rc.Code == code.String() {
			u.RecoveryCodes = append(u.RecoveryCodes[:i], u.RecoveryCodes[i+1:]...)
			u.LastLoggedInAt = time.Now()

			u.Events.Enqueue(AuthenticatedWithRecoveryCode{
				Email: u.Email,
			})

			return nil
		}
	}

	errs := errors.Map{"recovery code": errors.New("invalid recovery code")}

	return errs.Tracef(app.ErrInvalidInput)
}
