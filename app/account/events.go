package account

type InitialUserSignedUp struct {
	Email  string
	System string
	Method string
}

type Invited struct {
	Email  string
	System string
	Method string
}

type SignedUp struct {
	Email      string
	System     string
	Method     string
	IsVerified bool
}

type AlreadySignedUp struct {
	Email       string
	System      string
	Method      string
	HasPassword bool
}

type SignedIn struct {
	Email  string
	System string
	Method string
}

type Verified struct {
	Email string
}

type Activated struct {
	Email       string
	System      string
	Method      string
	HasPassword bool
}

type Suspended struct {
	Email  string
	Reason string
}

type SuspendedReasonChanged struct {
	Email  string
	Reason string
}

type Unsuspended struct {
	Email string
}

type TOTPDisabled struct {
	Email string
}

type TOTPResetRequested struct {
	Email string
}

type TOTPResetRequestApproved struct {
	Email string
}

type TOTPResetRequestDenied struct {
	Email string
}

type TOTPReset struct {
	Email string
}

type RecoveryCodesRegenerated struct {
	Email string
}

type TOTPTelChanged struct {
	Email  string
	OldTel string
	NewTel string
}

type PasswordChanged struct {
	Email string
}

type PasswordChosen struct {
	Email string
}

type PasswordReset struct {
	Email string
}

type RolesChanged struct {
	Email string
}
