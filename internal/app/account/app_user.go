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

const SignInMethodWebsite = "Website"

var ErrNotActivated = errors.New("account is not activated")

type User struct {
	aggregate.Root

	ID                 int
	Email              string
	HashedPassword     []byte
	TOTPMethod         string
	TOTPTelephone      string
	TOTPKey            []byte
	TOTPAlgorithm      string
	TOTPDigits         int
	TOTPPeriod         time.Duration
	TOTPVerifiedAt     time.Time
	TOTPActivatedAt    time.Time
	SignedUpAt         time.Time
	ActivatedAt        time.Time
	LastSignedInAt     time.Time
	LastSignedInMethod string
	Roles              []*Role
	RecoveryCodes      []*RecoveryCode
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
	return !u.TOTPVerifiedAt.IsZero()
}

func (u *User) HasActivatedTOTP() bool {
	return !u.TOTPActivatedAt.IsZero()
}

func (u *User) SignUp() error {
	u.SignedUpAt = time.Now().UTC()

	u.Events.Enqueue(SignedUp{
		Email: u.Email,
	})

	return nil
}

func (u *User) Activate(password Password, hasher password.Hasher) error {
	if !u.ActivatedAt.IsZero() {
		return errors.Tracef("already activated")
	}

	if err := u.setPassword(password, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.ActivatedAt = time.Now().UTC()

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
	if u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "TOTP already setup and activated")
	}

	key, err := NewTOTPKey(otp.SHA1)
	if err != nil {
		return errors.Tracef(err)
	}

	u.TOTPMethod = TOTPMethodNone.String()
	u.TOTPKey = key
	u.TOTPAlgorithm = otp.SHA1.String()
	u.TOTPDigits = 6
	u.TOTPPeriod = 30 * time.Second
	u.TOTPVerifiedAt = time.Time{}

	return nil
}

func (u *User) verifyTOTP(totp TOTP) error {
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

	return nil
}

func (u *User) VerifyTOTP(totp TOTP, method TOTPMethod) error {
	if u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "already verified and activated")
	}

	if err := u.verifyTOTP(totp); err != nil {
		return errors.Tracef(err)
	}

	u.TOTPMethod = method.String()
	u.TOTPVerifiedAt = time.Now().UTC()

	nCodes := 6
	u.RecoveryCodes = make([]*RecoveryCode, nCodes)

	for i := 0; i < nCodes; i++ {
		code, err := GenerateCode()
		if err != nil {
			return errors.Tracef(err)
		}

		u.RecoveryCodes[i] = NewRecoveryCode(code)
	}

	return nil
}

func (u *User) ActivateTOTP() error {
	if !u.HasVerifiedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "unverified TOTP cannot be activated")
	}

	if u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "already activated")
	}

	u.TOTPActivatedAt = time.Now().UTC()

	return nil
}

func (u *User) ChangeTOTPTelephone(newTelephone text.Tel) error {
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

func (u *User) RegenerateRecoveryCodes(totp TOTP) error {
	if !u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "cannot regenerate recovery codes without an activated TOTP")
	}

	if err := u.verifyTOTP(totp); err != nil {
		return errors.Tracef(err)
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
	if !u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "cannot disable an unactivated TOTP")
	}

	if _, err := u.SignInWithPassword(password, hasher); err != nil {
		return errors.Tracef(err)
	}

	u.TOTPMethod = TOTPMethodNone.String()
	u.TOTPTelephone = ""
	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.TOTPActivatedAt = time.Time{}
	u.RecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{
		Email: u.Email,
	})

	return nil
}

func (u *User) DisableTOTPWithRecoveryCode(code Code) error {
	if err := u.SignInWithRecoveryCode(code); err != nil {
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

func (u *User) SignInWithPassword(password Password, hasher password.Hasher) (bool, error) {
	if u.ActivatedAt.IsZero() {
		return false, errors.Tracef(app.ErrBadRequest, ErrNotActivated)
	}

	rehashed, err := u.verifyPassword(password, hasher)
	if err != nil {
		return false, errors.Tracef(err)
	}

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = time.Now().UTC()
		u.LastSignedInMethod = SignInMethodWebsite
	}

	u.Events.Enqueue(SignedInWithPassword{
		Email: u.Email,
	})

	return rehashed, nil
}

func (u *User) SignInWithTOTP(totp TOTP) error {
	if !u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "account does not have TOTP")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(app.ErrBadRequest, ErrNotActivated)
	}

	if err := u.verifyTOTP(totp); err != nil {
		return errors.Tracef(err)
	}

	u.LastSignedInAt = time.Now().UTC()

	u.Events.Enqueue(SignedInWithTOTP{
		Email: u.Email,
	})

	return nil
}

func (u *User) SignInWithRecoveryCode(code Code) error {
	if !u.HasActivatedTOTP() {
		return errors.Tracef(app.ErrBadRequest, "account cannot use recovery codes")
	}

	if u.ActivatedAt.IsZero() {
		return errors.Tracef(app.ErrBadRequest, ErrNotActivated)
	}

	for i, rc := range u.RecoveryCodes {
		if rc.Code == code.String() {
			u.RecoveryCodes = append(u.RecoveryCodes[:i], u.RecoveryCodes[i+1:]...)
			u.LastSignedInAt = time.Now().UTC()

			u.Events.Enqueue(SignedInWithRecoveryCode{
				Email: u.Email,
			})

			return nil
		}
	}

	errs := errors.Map{"recovery code": errors.New("invalid recovery code")}

	return errs.Tracef(app.ErrInvalidInput)
}
