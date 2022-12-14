package repo

import "github.com/polyscone/tofu/internal/pkg/errors"

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)
