package entity

import "github.com/polyscone/tofu/internal/pkg/errors"

type ID int

func NewID(id int) (ID, error) {
	if id <= 0 {
		return 0, errors.Tracef("id must be greater than or equal to 1")
	}

	return ID(id), nil
}
