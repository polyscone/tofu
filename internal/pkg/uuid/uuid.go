package uuid

import (
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var Nil UUID

type UUID [16]byte

func Parse(str string) (UUID, error) {
	if len(str) != 36 {
		return Nil, errors.New("UUID string must be 36 bytes in length")
	}
	if !validV7.MatchString(str) && !validV4.MatchString(str) {
		return Nil, errors.New("invalid UUID format")
	}
	if str == Nil.String() {
		return Nil, errors.New("UUID string is nil")
	}

	str = strings.ReplaceAll(str, "-", "")
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return Nil, fmt.Errorf("decode hex id string: %w", err)
	}

	return UUID(*(*[16]byte)(decoded)), nil
}

func (id UUID) IsNil() bool {
	return id == Nil
}

func (id UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}

func (id *UUID) Scan(src any) error {
	switch src := src.(type) {
	case nil:
		*id = Nil

	case string:
		u, err := Parse(src)
		if err != nil {
			return fmt.Errorf("parse %T: %w", src, err)
		}

		*id = u

	case []byte:
		u, err := Parse(string(src))
		if err != nil {
			return fmt.Errorf("parse %T: %w", src, err)
		}

		*id = u

	default:
		return fmt.Errorf("unable to scan %T into UUID", src)
	}

	return nil
}

func (id UUID) Value() (driver.Value, error) {
	if id.IsNil() {
		return nil, nil
	}

	return id.String(), nil
}
