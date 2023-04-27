package domain

type TOTPParams struct {
	Key       TOTPKey
	Algorithm string
	Digits    int
	Period    int
}
