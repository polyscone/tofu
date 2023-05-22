package errors

import "errors"

// Re-export standard library functions
var (
	As     = errors.As
	Is     = errors.Is
	Join   = errors.Join
	New    = errors.New
	Unwrap = errors.Unwrap
)
