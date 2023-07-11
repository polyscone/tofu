package account

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/otp"
)

const (
	SignInMethodNone   = ""
	SignInMethodForm   = "Form"
	SignInMethodGoogle = "Google"
)

var (
	ErrNotActivated    = errors.New("account is not activated")
	ErrInvalidPassword = errors.New("invalid password")
)

type User struct {
	aggregate.Root

	ID                   int
	Email                string
	HashedPassword       []byte
	TOTPMethod           string
	TOTPTel              string
	TOTPKey              []byte
	TOTPAlgorithm        string
	TOTPDigits           int
	TOTPPeriod           time.Duration
	TOTPVerifiedAt       time.Time
	TOTPActivatedAt      time.Time
	TOTPResetRequestedAt time.Time
	TOTPResetApprovedAt  time.Time
	SignedUpAt           time.Time
	ActivatedAt          time.Time
	LastSignedInAt       time.Time
	LastSignedInMethod   string
	HashedRecoveryCodes  [][]byte
	Roles                []*Role
	Grants               []string
	Denials              []string
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
		u.Events.Enqueue(AlreadySignedUp{Email: u.Email})

		return nil
	}

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = time.Now().UTC()
	}

	u.Events.Enqueue(SignedUp{Email: u.Email})

	return nil
}

func (u *User) SignUpWithGoogle() error {
	now := time.Now().UTC()

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = now
	}

	if u.ActivatedAt.IsZero() {
		u.ActivatedAt = now
	}

	u.Events.Enqueue(SignedUpWithGoogle{Email: u.Email})

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

	if _, err := u.checkPassword(oldPassword, hasher); err != nil {
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

func (u *User) ChoosePassword(newPassword Password, hasher Hasher) error {
	if u.ActivatedAt.IsZero() {
		return errors.New("cannot choose password until activated")
	}

	if len(u.HashedPassword) > 0 {
		return fmt.Errorf("cannot replace an already chosen password")
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.Events.Enqueue(PasswordChosen{Email: u.Email})

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

func (u *User) checkTOTP(totp TOTP) error {
	alg, err := otp.NewAlgorithm(u.TOTPAlgorithm)
	if err != nil {
		return fmt.Errorf("new OTP algorithm: %w", err)
	}

	tb, err := otp.NewTimeBased(u.TOTPDigits, alg, time.Unix(0, 0), u.TOTPPeriod)
	if err != nil {
		return fmt.Errorf("new time based OTP generator: %w", err)
	}

	ok, err := tb.Check(u.TOTPKey, time.Now(), 1, totp.String())
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

	if err := u.checkTOTP(totp); err != nil {
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

	if err := u.checkTOTP(totp); err != nil {
		return nil, fmt.Errorf("check TOTP: %w", err)
	}

	codes, err := u.replaceRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("replace recovery codes: %w", err)
	}

	u.Events.Enqueue(RecoveryCodesRegenerated{Email: u.Email})

	return codes, nil
}

func (u *User) disableTOTP() {
	u.TOTPMethod = TOTPMethodNone.String()
	u.TOTPTel = ""
	u.TOTPKey = nil
	u.TOTPAlgorithm = ""
	u.TOTPDigits = 0
	u.TOTPPeriod = 0
	u.TOTPVerifiedAt = time.Time{}
	u.TOTPActivatedAt = time.Time{}
	u.HashedRecoveryCodes = nil
}

func (u *User) DisableTOTP(password Password, hasher Hasher) error {
	if !u.HasActivatedTOTP() {
		return errors.New("cannot disable an unactivated TOTP")
	}

	if _, err := u.checkPassword(password, hasher); err != nil {
		return fmt.Errorf("check password: %w", err)
	}

	u.disableTOTP()

	u.Events.Enqueue(TOTPDisabled{Email: u.Email})

	return nil
}

func (u *User) ResetTOTP(password Password, hasher Hasher) error {
	if !u.HasActivatedTOTP() {
		return errors.New("cannot reset an unactivated TOTP")
	}

	if u.TOTPResetApprovedAt.IsZero() {
		return errors.New("cannot approve a TOTP reset request that is still awaiting review")
	}

	if _, err := u.checkPassword(password, hasher); err != nil {
		return fmt.Errorf("check password: %w", err)
	}

	u.disableTOTP()

	u.TOTPResetApprovedAt = time.Time{}

	u.Events.Enqueue(TOTPReset{Email: u.Email})

	return nil
}

func (u *User) RequestTOTPReset() error {
	if !u.HasActivatedTOTP() {
		return errors.New("cannot request a reset for an unactivated TOTP")
	}

	u.TOTPResetRequestedAt = time.Now().UTC()

	u.Events.Enqueue(TOTPResetRequested{Email: u.Email})

	return nil
}

func (u *User) ApproveTOTPResetRequest() error {
	if !u.HasActivatedTOTP() {
		return errors.New("cannot request a reset for an unactivated TOTP")
	}

	u.TOTPResetRequestedAt = time.Time{}
	u.TOTPResetApprovedAt = time.Now().UTC()

	u.Events.Enqueue(TOTPResetRequestApproved{Email: u.Email})

	return nil
}

func (u *User) DenyTOTPResetRequest() error {
	if u.TOTPResetRequestedAt.IsZero() {
		return errors.New("cannot deny a non-existent TOTP reset request")
	}

	u.TOTPResetRequestedAt = time.Time{}

	u.Events.Enqueue(TOTPResetRequestDenied{Email: u.Email})

	return nil
}

func (u *User) checkPassword(password Password, hasher Hasher) (rehashed bool, _ error) {
	ok, rehash, err := hasher.CheckPasswordHash(password.data, u.HashedPassword)
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

func (u *User) SignInWithPassword(password Password, hasher Hasher) (rehashed bool, _ error) {
	if u.ActivatedAt.IsZero() {
		// Always check a password even when we error finding a user to help
		// avoid leaking info that would allow enumeration of valid emails
		if err := hasher.CheckDummyPasswordHash(); err != nil {
			return false, fmt.Errorf("check dummy password hash: %w", err)
		}

		return false, ErrNotActivated
	}

	rehashed, err := u.checkPassword(password, hasher)
	if err != nil {
		return false, fmt.Errorf("check password: %w", err)
	}

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = time.Now().UTC()
		u.LastSignedInMethod = SignInMethodForm
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

	if err := u.checkTOTP(totp); err != nil {
		return fmt.Errorf("check TOTP: %w", err)
	}

	u.LastSignedInAt = time.Now().UTC()

	u.Events.Enqueue(SignedInWithTOTP{Email: u.Email})

	return nil
}

func (u *User) useRecoveryCode(code RecoveryCode) error {
	for i, rc := range u.HashedRecoveryCodes {
		if code.EqualHash(rc) {
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

func (u *User) SignInWithGoogle() error {
	if u.ActivatedAt.IsZero() {
		return ErrNotActivated
	}

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = time.Now().UTC()
		u.LastSignedInMethod = SignInMethodGoogle
	}

	u.Events.Enqueue(SignedInWithGoogle{Email: u.Email})

	return nil
}

func (u *User) ChangeRoles(roles []*Role, grants, denials []Permission) error {
	if u.ActivatedAt.IsZero() {
		return errors.New("cannot change roles until activated")
	}

	var containsSuper bool
	for _, role := range roles {
		if containsSuper = role.ID == SuperRole.ID; containsSuper {
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
