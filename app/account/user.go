package account

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/aggregate"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/internal/otp"
)

const (
	SignInMethodNone      = ""
	SignInMethodPassword  = "account.user.sign_in_method.password"
	SignInMethodMagicLink = "account.user.sign_in_method.magic_link"
	SignInMethodGoogle    = "account.user.sign_in_method.google"
	SignInMethodFacebook  = "account.user.sign_in_method.facebook"
)

const (
	SignUpMethodNone        = ""
	SignUpMethodSystemSetup = "account.user.sign_up_method.system_setup"
	SignUpMethodWebForm     = "account.user.sign_up_method.web_form"
	SignUpMethodMagicLink   = "account.user.sign_up_method.magic_link"
	SignUpMethodGoogle      = "account.user.sign_up_method.google"
	SignUpMethodFacebook    = "account.user.sign_up_method.facebook"
	SignUpMethodInvite      = "account.user.sign_up_method.invite"
)

var (
	ErrNotVerified      = i18n.M("account.user.error.not_verified")
	ErrAlreadyVerified  = i18n.M("account.user.error.already_verified")
	ErrNotActivated     = i18n.M("account.user.error.not_activated")
	ErrAlreadyActivated = i18n.M("account.user.error.already_activated")
	ErrSuspended        = i18n.M("account.user.error.is_suspended")
	ErrInvalidPassword  = i18n.M("account.user.error.invalid_password")
)

type User struct {
	aggregate.Root

	ID                      int
	Email                   string
	HashedPassword          []byte
	TOTPMethod              string
	TOTPTel                 string
	TOTPKey                 []byte
	TOTPAlgorithm           string
	TOTPDigits              int
	TOTPPeriod              time.Duration
	TOTPVerifiedAt          time.Time
	TOTPActivatedAt         time.Time
	TOTPResetRequestedAt    time.Time
	TOTPResetApprovedAt     time.Time
	InvitedAt               time.Time
	SignedUpAt              time.Time
	SignedUpSystem          string
	SignedUpMethod          string
	VerifiedAt              time.Time
	ActivatedAt             time.Time
	LastSignInAttemptAt     time.Time
	LastSignInAttemptSystem string
	LastSignInAttemptMethod string
	LastSignedInAt          time.Time
	LastSignedInSystem      string
	LastSignedInMethod      string
	SuspendedAt             time.Time
	SuspendedReason         string
	HashedRecoveryCodes     [][]byte
	Roles                   []*Role
	Grants                  []string
	Denials                 []string
}

type UserFilter struct {
	ID     *int
	Email  *string
	Search *string
	RoleID *int

	SortTopID int
	Sorts     []string

	Limit  int
	Offset int
}

func NewUser(email Email) *User {
	return &User{Email: email.String()}
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

func (u *User) HasSetupTOTP() bool {
	return len(u.TOTPKey) > 0 &&
		u.TOTPAlgorithm != "" &&
		u.TOTPDigits > 0 &&
		u.TOTPPeriod > 0
}

func (u *User) HasVerifiedTOTP() bool {
	return !u.TOTPVerifiedAt.IsZero()
}

func (u *User) HasActivatedTOTP() bool {
	return !u.TOTPActivatedAt.IsZero()
}

func (u *User) IsVerified() bool {
	return !u.VerifiedAt.IsZero()
}

func (u *User) IsActivated() bool {
	return !u.ActivatedAt.IsZero()
}

func (u *User) IsSuspended() bool {
	return !u.SuspendedAt.IsZero()
}

func (u *User) HasSignedIn() bool {
	return !u.LastSignedInAt.IsZero()
}

func (u *User) Invite(system string) error {
	if !u.VerifiedAt.IsZero() {
		return i18n.M("invite:account.user.error.already_verified")
	}

	if u.InvitedAt.IsZero() {
		u.InvitedAt = time.Now().UTC()
	}

	u.SignedUpSystem = system
	u.SignedUpMethod = SignUpMethodInvite

	u.Events.Enqueue(Invited{
		Email:  u.Email,
		System: system,
		Method: SignUpMethodInvite,
	})

	return nil
}

func (u *User) SignUpAsInitialUser(system string, roles []*Role, password Password, hasher Hasher) error {
	if !u.SignedUpAt.IsZero() {
		return i18n.M("initial_user_sign_up:account.user.error.already_signed_up")
	}
	if !u.VerifiedAt.IsZero() {
		return i18n.M("initial_user_sign_up:account.user.error.already_verified")
	}
	if !u.ActivatedAt.IsZero() {
		return i18n.M("initial_user_sign_up:account.user.error.already_activated")
	}

	if err := u.setPassword(password, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	now := time.Now().UTC()

	u.SignedUpAt = now
	u.SignedUpSystem = system
	u.SignedUpMethod = SignUpMethodSystemSetup

	u.VerifiedAt = now
	u.ActivatedAt = now

	u.Roles = roles

	u.Events.Enqueue(InitialUserSignedUp{
		Email:  u.Email,
		System: system,
		Method: SignUpMethodSystemSetup,
	})

	return nil
}

func (u *User) SignUp(system string) {
	if !u.ActivatedAt.IsZero() {
		u.Events.Enqueue(AlreadySignedUp{
			Email:       u.Email,
			System:      system,
			Method:      u.SignedUpMethod,
			HasPassword: len(u.HashedPassword) != 0,
		})

		return
	}

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = time.Now().UTC()
	}

	u.SignedUpSystem = system
	u.SignedUpMethod = SignUpMethodWebForm

	u.Events.Enqueue(SignedUp{
		Email:      u.Email,
		System:     system,
		Method:     SignUpMethodWebForm,
		IsVerified: !u.VerifiedAt.IsZero(),
	})
}

func (u *User) SignUpWithMagicLink(system string) {
	if !u.ActivatedAt.IsZero() {
		return
	}

	now := time.Now().UTC()

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = now
	}

	if u.VerifiedAt.IsZero() {
		u.VerifiedAt = now
	}

	u.SignedUpSystem = system
	u.SignedUpMethod = SignUpMethodMagicLink

	u.Events.Enqueue(SignedUp{
		Email:      u.Email,
		System:     system,
		Method:     SignUpMethodMagicLink,
		IsVerified: !u.VerifiedAt.IsZero(),
	})
}

func (u *User) SignUpWithGoogle(system string) {
	if !u.ActivatedAt.IsZero() {
		return
	}

	now := time.Now().UTC()

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = now
	}

	if u.VerifiedAt.IsZero() {
		u.VerifiedAt = now
	}

	u.SignedUpSystem = system
	u.SignedUpMethod = SignUpMethodGoogle

	u.Events.Enqueue(SignedUp{
		Email:      u.Email,
		System:     system,
		Method:     SignUpMethodGoogle,
		IsVerified: !u.VerifiedAt.IsZero(),
	})
}

func (u *User) SignUpWithFacebook(system string) {
	if !u.ActivatedAt.IsZero() {
		return
	}

	now := time.Now().UTC()

	if u.SignedUpAt.IsZero() {
		u.SignedUpAt = now
	}

	if u.VerifiedAt.IsZero() {
		u.VerifiedAt = now
	}

	u.SignedUpSystem = system
	u.SignedUpMethod = SignUpMethodFacebook

	u.Events.Enqueue(SignedUp{
		Email:      u.Email,
		System:     system,
		Method:     SignUpMethodFacebook,
		IsVerified: !u.VerifiedAt.IsZero(),
	})
}

func (u *User) Verify(password Password, hasher Hasher) error {
	if !u.VerifiedAt.IsZero() {
		return ErrAlreadyVerified.WithContext("verify")
	}

	if err := u.setPassword(password, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.VerifiedAt = time.Now().UTC()

	u.Events.Enqueue(Verified{Email: u.Email})

	return nil
}

func (u *User) Activate() error {
	if u.VerifiedAt.IsZero() {
		return ErrNotVerified.WithContext("activate")
	}

	if !u.ActivatedAt.IsZero() {
		return ErrAlreadyActivated.WithContext("activate")
	}

	u.ActivatedAt = time.Now().UTC()

	u.Events.Enqueue(Activated{
		Email:       u.Email,
		System:      u.SignedUpSystem,
		Method:      u.SignedUpMethod,
		HasPassword: len(u.HashedPassword) != 0,
	})

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
	if u.VerifiedAt.IsZero() {
		return i18n.M("change_password:account.user.error.not_verified")
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
		return i18n.M("choose_password:account.user.error.not_activated")
	}

	if len(u.HashedPassword) != 0 {
		return i18n.M("choose_password:account.user.error.already_has_password")
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.Events.Enqueue(PasswordChosen{Email: u.Email})

	return nil
}

func (u *User) ResetPassword(newPassword Password, hasher Hasher) error {
	if u.ActivatedAt.IsZero() {
		return i18n.M("reset_password:account.user.error.not_activated")
	}

	if err := u.setPassword(newPassword, hasher); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	u.Events.Enqueue(PasswordReset{Email: u.Email})

	return nil
}

func (u *User) SetupTOTP() error {
	if u.ActivatedAt.IsZero() {
		return i18n.M("setup_totp:account.user.error.not_activated")
	}

	if u.HasActivatedTOTP() {
		return i18n.M("setup_totp:account.user.error.totp_already_activated")
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
				"totp": i18n.M("check_totp:account.user.error.totp_already_used"),
			})
		}

		return err
	}
	if !ok {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"totp": i18n.M("check_totp:account.user.error.invalid_totp"),
		})
	}

	return nil
}

func (u *User) VerifyTOTP(totp TOTP, method TOTPMethod) ([]string, error) {
	if u.HasActivatedTOTP() {
		return nil, i18n.M("verify_totp:account.user.error.totp_already_activated")
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
		return i18n.M("activate_totp:account.user.error.totp_not_verified")
	}

	if u.HasActivatedTOTP() {
		return i18n.M("activate_totp:account.user.error.totp_already_activated")
	}

	u.TOTPActivatedAt = time.Now().UTC()

	return nil
}

func (u *User) Suspend(reason SuspendedReason) {
	if u.IsSuspended() {
		if u.SuspendedReason != reason.String() {
			u.SuspendedReason = reason.String()

			u.Events.Enqueue(SuspendedReasonChanged{
				Email:  u.Email,
				Reason: u.SuspendedReason,
			})
		}

		return
	}

	u.SuspendedAt = time.Now().UTC()
	u.SuspendedReason = reason.String()

	u.Events.Enqueue(Suspended{
		Email:  u.Email,
		Reason: u.SuspendedReason,
	})
}

func (u *User) Unsuspend() {
	u.SuspendedReason = ""

	if u.IsSuspended() {
		u.SuspendedAt = time.Time{}

		u.Events.Enqueue(Unsuspended{Email: u.Email})
	}
}

func (u *User) ChangeTOTPTel(newTel Tel) error {
	if len(u.TOTPKey) == 0 {
		return i18n.M("change_totp_tel:account.user.error.no_totp_key")
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
		return "", i18n.M("generate_totp:account.user.error.no_totp_key")
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

	for i := range nCodes {
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
		return nil, i18n.M("regenerate_recovery_codes:account.user.error.totp_not_activated")
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
		return i18n.M("disable_totp:account.user.error.totp_not_activated")
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
		return i18n.M("reset_totp:account.user.error.totp_not_activated")
	}

	if u.TOTPResetApprovedAt.IsZero() {
		return i18n.M("reset_totp:account.user.error.totp_reset_not_approved")
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
		return i18n.M("request_totp_reset:account.user.error.totp_not_activated")
	}

	u.TOTPResetRequestedAt = time.Now().UTC()

	u.Events.Enqueue(TOTPResetRequested{Email: u.Email})

	return nil
}

func (u *User) ApproveTOTPResetRequest() error {
	if !u.HasActivatedTOTP() {
		return i18n.M("approve_totp_reset_request:account.user.error.totp_not_activated")
	}

	u.TOTPResetRequestedAt = time.Time{}
	u.TOTPResetApprovedAt = time.Now().UTC()

	u.Events.Enqueue(TOTPResetRequestApproved{Email: u.Email})

	return nil
}

func (u *User) DenyTOTPResetRequest() error {
	if u.TOTPResetRequestedAt.IsZero() {
		return i18n.M("deny_totp_reset_request:account.user.error.totp_reset_not_requested")
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

func (u *User) SignInWithPassword(system string, password Password, hasher Hasher) (bool, error) {
	if u.VerifiedAt.IsZero() || u.ActivatedAt.IsZero() || u.IsSuspended() {
		// Always check a password even in error cases to help
		// avoid leaking info that would allow enumeration of valid emails
		if err := hasher.CheckDummyPasswordHash(); err != nil {
			return false, fmt.Errorf("check dummy password hash: %w", err)
		}

		if u.VerifiedAt.IsZero() {
			return false, ErrNotVerified.WithContext("sign_in_with_password")
		}

		if u.ActivatedAt.IsZero() {
			return false, ErrNotActivated.WithContext("sign_in_with_password")
		}

		return false, ErrSuspended.WithContext("sign_in_with_password")
	}

	rehashed, err := u.checkPassword(password, hasher)
	if err != nil {
		if msg, ok := err.(i18n.Message); ok {
			err = msg.WithContext("sign_in_with_password")
		}

		return false, fmt.Errorf("check password: %w", err)
	}

	u.LastSignInAttemptAt = time.Now().UTC()
	u.LastSignInAttemptSystem = system
	u.LastSignInAttemptMethod = SignInMethodPassword

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = u.LastSignInAttemptAt
		u.LastSignedInSystem = u.LastSignInAttemptSystem
		u.LastSignedInMethod = u.LastSignInAttemptMethod
	}

	u.Events.Enqueue(SignedIn{
		Email:  u.Email,
		System: system,
		Method: SignInMethodPassword,
	})

	return rehashed, nil
}

func (u *User) SignInWithMagicLink(system string) error {
	if u.VerifiedAt.IsZero() {
		return ErrNotVerified.WithContext("sign_in_with_magic_link")
	}

	if u.ActivatedAt.IsZero() {
		return ErrNotActivated.WithContext("sign_in_with_magic_link")
	}

	if u.IsSuspended() {
		return ErrSuspended.WithContext("sign_in_with_magic_link")
	}

	u.LastSignInAttemptAt = time.Now().UTC()
	u.LastSignInAttemptSystem = system
	u.LastSignInAttemptMethod = SignInMethodMagicLink

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = u.LastSignInAttemptAt
		u.LastSignedInSystem = u.LastSignInAttemptSystem
		u.LastSignedInMethod = u.LastSignInAttemptMethod
	}

	u.Events.Enqueue(SignedIn{
		Email:  u.Email,
		System: system,
		Method: SignInMethodMagicLink,
	})

	return nil
}

func (u *User) SignInWithTOTP(system string, totp TOTP) error {
	if u.ActivatedAt.IsZero() {
		return ErrNotActivated.WithContext("sign_in_with_totp")
	}

	if !u.HasActivatedTOTP() {
		return i18n.M("sign_in_with_totp:account.user.error.totp_not_activated")
	}

	if err := u.checkTOTP(totp); err != nil {
		return fmt.Errorf("check TOTP: %w", err)
	}

	if u.IsSuspended() {
		return ErrSuspended.WithContext("sign_in_with_totp")
	}

	u.LastSignInAttemptAt = time.Now().UTC()
	u.LastSignedInAt = u.LastSignInAttemptAt
	u.LastSignedInSystem = system
	u.LastSignedInMethod = u.LastSignInAttemptMethod

	u.Events.Enqueue(SignedIn{
		Email:  u.Email,
		System: system,
		Method: u.LastSignedInMethod,
	})

	return nil
}

func (u *User) useRecoveryCode(code RecoveryCode) error {
	n := len(u.HashedRecoveryCodes)
	u.HashedRecoveryCodes = slices.DeleteFunc(u.HashedRecoveryCodes, func(rc []byte) bool {
		return code.EqualHash(rc)
	})

	deleted := n - len(u.HashedRecoveryCodes)
	switch {
	case deleted == 0:
		return fmt.Errorf("unknown recovery code: %v", code)

	case deleted > 1:
		return fmt.Errorf("multiple recovery codes with the value %v removed", code)

	case deleted < 0:
		return fmt.Errorf("recovery code with the value %v was added instead of removed", code)
	}

	return nil
}

func (u *User) SignInWithRecoveryCode(system string, code RecoveryCode) error {
	if u.IsSuspended() {
		return ErrSuspended.WithContext("sign_in_with_recovery_code")
	}

	if u.ActivatedAt.IsZero() {
		return ErrNotActivated.WithContext("sign_in_with_recovery_code")
	}

	if !u.HasActivatedTOTP() {
		return i18n.M("sign_in_with_recovery_code:account.user.error.totp_not_activated")
	}

	if err := u.useRecoveryCode(code); err != nil {
		return fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
			"recovery code": i18n.M("sign_in_with_recovery_code:account.user.error.invalid_recovery_code"),
		})
	}

	u.LastSignInAttemptAt = time.Now().UTC()
	u.LastSignedInAt = u.LastSignInAttemptAt
	u.LastSignedInSystem = system
	u.LastSignedInMethod = u.LastSignInAttemptMethod

	u.Events.Enqueue(SignedIn{
		Email:  u.Email,
		System: system,
		Method: u.LastSignedInMethod,
	})

	return nil
}

func (u *User) SignInWithGoogle(system string) error {
	if u.VerifiedAt.IsZero() {
		return ErrNotVerified.WithContext("sign_in_with_google")
	}

	if u.ActivatedAt.IsZero() {
		return ErrNotActivated.WithContext("sign_in_with_google")
	}

	if u.IsSuspended() {
		return ErrSuspended.WithContext("sign_in_with_google")
	}

	u.LastSignInAttemptAt = time.Now().UTC()
	u.LastSignInAttemptSystem = system
	u.LastSignInAttemptMethod = SignInMethodGoogle

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = u.LastSignInAttemptAt
		u.LastSignedInSystem = u.LastSignInAttemptSystem
		u.LastSignedInMethod = u.LastSignInAttemptMethod
	}

	u.Events.Enqueue(SignedIn{
		Email:  u.Email,
		System: system,
		Method: SignInMethodGoogle,
	})

	return nil
}

func (u *User) SignInWithFacebook(system string) error {
	if u.VerifiedAt.IsZero() {
		return ErrNotVerified.WithContext("sign_in_with_facebook")
	}

	if u.ActivatedAt.IsZero() {
		return ErrNotActivated.WithContext("sign_in_with_facebook")
	}

	if u.IsSuspended() {
		return ErrSuspended.WithContext("sign_in_with_facebook")
	}

	u.LastSignInAttemptAt = time.Now().UTC()
	u.LastSignInAttemptSystem = system
	u.LastSignInAttemptMethod = SignInMethodFacebook

	if !u.HasActivatedTOTP() {
		u.LastSignedInAt = u.LastSignInAttemptAt
		u.LastSignedInSystem = u.LastSignInAttemptSystem
		u.LastSignedInMethod = u.LastSignInAttemptMethod
	}

	u.Events.Enqueue(SignedIn{
		Email:  u.Email,
		System: system,
		Method: SignInMethodFacebook,
	})

	return nil
}

func (u *User) ChangeRoles(roles []*Role, grants, denials []Permission) {
	u.Roles = roles

	u.Grants = nil
GrantLoop:
	for _, grant := range grants {
		for _, denial := range denials {
			if grant == denial {
				continue GrantLoop
			}
		}

		u.Grants = append(u.Grants, grant.String())
	}

	u.Denials = nil
	if denials != nil {
		u.Denials = make([]string, len(denials))

		for i, denial := range denials {
			u.Denials[i] = denial.String()
		}
	}

	u.Events.Enqueue(RolesChanged{Email: u.Email})
}
