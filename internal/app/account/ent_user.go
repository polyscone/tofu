package account

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/otp"
)

const SignInMethodWebsite = "Website"

var (
	ErrNotActivated    = errors.New("account is not activated")
	ErrInvalidPassword = errors.New("invalid password")
)

type User struct {
	aggregate.Root

	ID                  int
	Email               string
	HashedPassword      []byte
	TOTPMethod          string
	TOTPTel             string
	TOTPKey             []byte
	TOTPAlgorithm       string
	TOTPDigits          int
	TOTPPeriod          time.Duration
	TOTPVerifiedAt      time.Time
	TOTPActivatedAt     time.Time
	SignedUpAt          time.Time
	ActivatedAt         time.Time
	LastSignedInAt      time.Time
	LastSignedInMethod  string
	HashedRecoveryCodes [][]byte
	Roles               []*Role
	Grants              []string
	Denials             []string
}

type UserFilter struct {
	ID     *int
	Email  *string
	Search *string
	RoleID *int

	SortTopID int

	Limit  int
	Offset int
}

func NewUser(email Email) *User {
	return &User{Email: email.String()}
}

func (u *User) IsSuper() bool {
	for _, role := range u.Roles {
		if role.ID == SuperRole.ID {
			return true
		}
	}

	return false
}

func (u *User) Permissions() []string {
	var permissions []string

	for _, role := range u.Roles {
	PermissionLoop:
		for _, permission := range role.Permissions {
			for _, denial := range u.Denials {
				if permission == denial {
					continue PermissionLoop
				}
			}

			permissions = append(permissions, permission)
		}
	}

GrantLoop:
	for _, grant := range u.Grants {
		for _, denial := range u.Denials {
			if grant == denial {
				continue GrantLoop
			}
		}

		permissions = append(permissions, grant)
	}

	return permissions
}

func (u *User) HasVerifiedTOTP() bool {
	return !u.TOTPVerifiedAt.IsZero()
}

func (u *User) HasActivatedTOTP() bool {
	return !u.TOTPActivatedAt.IsZero()
}

func (u *User) SignUp() error {
	if !u.ActivatedAt.IsZero() {
		return nil
	}

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = time.Now().UTC()
	}

	u.Events.Enqueue(SignedUp{Email: u.Email})

	return nil
}

func (u *User) Activate(password Password, hasher Hasher) error {
	if !u.ActivatedAt.IsZero() {
		return errors.New("already activated")
	}

	if err := u.setPassword(password, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.ActivatedAt = time.Now().UTC()

	u.Events.Enqueue(Activated{Email: u.Email})

	return nil
}

func (u *User) setPassword(newPassword Password, hasher Hasher) error {
	hashedPassword, err := hasher.EncodedPasswordHash(newPassword.data)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	u.HashedPassword = hashedPassword

	return nil
}

func (u *User) ChangePassword(oldPassword, newPassword Password, hasher Hasher) error {
	if u.ActivatedAt.IsZero() {
		return errors.New("cannot change password until activated")
	}

	if _, err := u.verifyPassword(oldPassword, hasher); err != nil {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"old password": err,
		})
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.Events.Enqueue(PasswordChanged{Email: u.Email})

	return nil
}

func (u *User) ResetPassword(newPassword Password, hasher Hasher) error {
	if u.ActivatedAt.IsZero() {
		return errors.New("cannot change password until activated")
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.Events.Enqueue(PasswordReset{Email: u.Email})

	return nil
}

func (u *User) SetupTOTP() error {
	if u.HasActivatedTOTP() {
		return errors.New("TOTP already setup and activated")
	}

	key, err := NewTOTPKey(otp.SHA1)
	if err != nil {
		return fmt.Errorf("new TOTP key: %w", err)
	}

	u.TOTPMethod = TOTPMethodNone.String()
	u.TOTPKey = key.data
	u.TOTPAlgorithm = otp.SHA1.String()
	u.TOTPDigits = 6
	u.TOTPPeriod = 30 * time.Second
	u.TOTPVerifiedAt = time.Time{}

	return nil
}

func (u *User) verifyTOTP(totp TOTP) error {
	alg, err := otp.NewAlgorithm(u.TOTPAlgorithm)
	if err != nil {
		return fmt.Errorf("new OTP algorithm: %w", err)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, alg, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return fmt.Errorf("new time based OTP generator: %w", err)
	}

	ok, err := tb.Verify(u.TOTPKey, time.Now(), 1, totp.String())
	if err != nil {
		if errors.Is(err, otp.ErrPasswordUsed) {
			return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
				"totp": errors.New("passcode already used"),
			})
		}

		return err
	}
	if !ok {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"totp": errors.New("invalid passcode"),
		})
	}

	return nil
}

func (u *User) VerifyTOTP(totp TOTP, method TOTPMethod) ([]string, error) {
	if u.HasActivatedTOTP() {
		return nil, errors.New("already verified and activated")
	}

	if err := u.verifyTOTP(totp); err != nil {
		return nil, err
	}

	u.TOTPMethod = method.String()
	u.TOTPVerifiedAt = time.Now().UTC()

	codes, err := u.replaceRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("replace recovery codes: %w", err)
	}

	return codes, nil
}

func (u *User) ActivateTOTP() error {
	if !u.HasVerifiedTOTP() {
		return errors.New("unverified TOTP cannot be activated")
	}

	if u.HasActivatedTOTP() {
		return errors.New("already activated")
	}

	u.TOTPActivatedAt = time.Now().UTC()

	return nil
}

func (u *User) ChangeTOTPTel(newTel Tel) error {
	if len(u.TOTPKey) == 0 {
		return errors.New("cannot change TOTP phone without a key setup")
	}

	if u.TOTPTel == newTel.String() {
		return nil
	}

	oldTel := u.TOTPTel

	u.TOTPTel = newTel.String()

	u.Events.Enqueue(TOTPTelChanged{
		Email:  u.Email,
		OldTel: oldTel,
		NewTel: u.TOTPTel,
	})

	return nil
}

func (u *User) GenerateTOTP() (string, error) {
	if len(u.TOTPKey) == 0 {
		return "", errors.New("cannot generate a TOTP without a key setup")
	}

	alg, err := otp.NewAlgorithm(u.TOTPAlgorithm)
	if err != nil {
		return "", fmt.Errorf("new OTP algorithm: %w", err)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, alg, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return "", fmt.Errorf("new time based OTP generator: %w", err)
	}

	totp, err := tb.Generate(u.TOTPKey, time.Now())
	if err != nil {
		return "", fmt.Errorf("generate TOTP: %w", err)
	}

	return totp, nil
}

func (u *User) replaceRecoveryCodes() ([]string, error) {
	nCodes := 6
	codes := make([]string, nCodes)
	hashedCodes := make([][]byte, nCodes)

	for i := 0; i < nCodes; i++ {
		code, err := NewRandomRecoveryCode()
		if err != nil {
			return nil, fmt.Errorf("new random recovery code: %w", err)
		}

		sum := sha256.Sum256([]byte(code))

		codes[i] = code.String()
		hashedCodes[i] = sum[:]
	}

	u.HashedRecoveryCodes = hashedCodes

	return codes, nil
}

func (u *User) RegenerateRecoveryCodes(totp TOTP) ([]string, error) {
	if !u.HasActivatedTOTP() {
		return nil, errors.New("cannot regenerate recovery codes without an activated TOTP")
	}

	if err := u.verifyTOTP(totp); err != nil {
		return nil, fmt.Errorf("verify TOTP: %w", err)
	}

	codes, err := u.replaceRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("replace recovery codes: %w", err)
	}

	u.Events.Enqueue(RecoveryCodesRegenerated{Email: u.Email})

	return codes, nil
}

func (u *User) DisableTOTP(password Password, hasher Hasher) error {
	if !u.HasActivatedTOTP() {
		return errors.New("cannot disable an unactivated TOTP")
	}

	if _, err := u.verifyPassword(password, hasher); err != nil {
		return fmt.Errorf("verify password: %w", err)
	}

	u.TOTPMethod = TOTPMethodNone.String()
	u.TOTPTel = ""
	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.TOTPActivatedAt = time.Time{}
	u.HashedRecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{Email: u.Email})

	return nil
}

func (u *User) DisableTOTPWithRecoveryCode(code RecoveryCode) error {
	if err := u.useRecoveryCode(code); err != nil {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"recovery code": errors.New("invalid recovery code"),
		})
	}

	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.HashedRecoveryCodes = nil

	u.Events.Enqueue(DisabledTOTP{Email: u.Email})

	return nil
}

func (u *User) verifyPassword(password Password, hasher Hasher) (rehash bool, _ error) {
	ok, rehash, err := hasher.VerifyPasswordHash(password.data, u.HashedPassword)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, ErrInvalidPassword
	}
	if rehash {
		if err := u.setPassword(password, hasher); err != nil {
			return true, fmt.Errorf("rehash password: %w", err)
		}

		return true, nil
	}

	return false, nil
}

func (u *User) SignInWithPassword(password Password, hasher Hasher) (bool, error) {
	if u.ActivatedAt.IsZero() {
		return false, ErrNotActivated
	}

	rehashed, err := u.verifyPassword(password, hasher)
	if err != nil {
		return false, fmt.Errorf("verify password: %w", err)
	}

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = time.Now().UTC()
		u.LastSignedInMethod = SignInMethodWebsite
	}

	u.Events.Enqueue(SignedInWithPassword{Email: u.Email})

	return rehashed, nil
}

func (u *User) SignInWithTOTP(totp TOTP) error {
	if !u.HasActivatedTOTP() {
		return errors.New("account does not have TOTP")
	}

	if u.ActivatedAt.IsZero() {
		return ErrNotActivated
	}

	if err := u.verifyTOTP(totp); err != nil {
		return fmt.Errorf("verify TOTP: %w", err)
	}

	u.LastSignedInAt = time.Now().UTC()

	u.Events.Enqueue(SignedInWithTOTP{Email: u.Email})

	return nil
}

func (u *User) useRecoveryCode(code RecoveryCode) error {
	sum := sha256.Sum256([]byte(code))
	hash := sum[:]

	for i, rc := range u.HashedRecoveryCodes {
		if bytes.Equal(rc, hash) {
			u.HashedRecoveryCodes = append(u.HashedRecoveryCodes[:i], u.HashedRecoveryCodes[i+1:]...)

			return nil
		}
	}

	return fmt.Errorf("unknown recovery code: %v", code)
}

func (u *User) SignInWithRecoveryCode(code RecoveryCode) error {
	if !u.HasActivatedTOTP() {
		return errors.New("account cannot use recovery codes")
	}

	if u.ActivatedAt.IsZero() {
		return ErrNotActivated
	}

	if err := u.useRecoveryCode(code); err != nil {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"recovery code": errors.New("invalid recovery code"),
		})
	}

	u.LastSignedInAt = time.Now().UTC()

	u.Events.Enqueue(SignedInWithRecoveryCode{Email: u.Email})

	return nil
}

func (u *User) ChangeRoles(roles []*Role, grants, denials []Permission) error {
	if u.ActivatedAt.IsZero() {
		return errors.New("cannot change roles until activated")
	}

	var containsSuper bool
	for _, role := range roles {
		if role.ID == SuperRole.ID {
			containsSuper = true

			break
		}
	}

	if u.IsSuper() && !containsSuper {
		return errors.New("cannot remove super role")
	}

	if containsSuper {
		grants = nil
		denials = nil
	}

	u.Roles = roles

	u.Grants = nil
	if grants != nil {
	GrantLoop:
		for _, grant := range grants {
			for _, denial := range denials {
				if grant == denial {
					continue GrantLoop
				}
			}

			u.Grants = append(u.Grants, grant.String())
		}
	}

	u.Denials = nil
	if denials != nil {
		u.Denials = make([]string, len(denials))

		for i, denial := range denials {
			u.Denials[i] = denial.String()
		}
	}

	u.Events.Enqueue(RolesChanged{Email: u.Email})

	return nil
}
