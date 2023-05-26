package account

type RecoveryCode struct {
	Code string
}

type RecoveryCodeFilter struct {
	UserID *int
}

func NewRecoveryCode(code Code) *RecoveryCode {
	return &RecoveryCode{Code: code.String()}
}
