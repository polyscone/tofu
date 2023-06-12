package uuid

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// validV4 is a regexp that matches any valid non-nil V4 UUID.
var validV4 = regexp.MustCompile("(?i)^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$")

// Nil represents the nil UUID as laid out in RFC4122.
var Nil V4

// V4 represents the byte data that a V4 UUID is made up of.
type V4 [16]byte

// NewV4 will create and return a new V4 UUID.
func NewV4() (V4, error) {
	var id V4

	rn, err := io.ReadFull(rand.Reader, id[:])
	if err != nil {
		return Nil, fmt.Errorf("read random bytes: %w", err)
	}
	if n := len(id); rn != n {
		return Nil, fmt.Errorf("could only read %v of %v bytes", rn, n)
	}

	id[6] = (id[6] & 0x0F) | (0x04 << 4)    // Set version to 4
	id[8] = (id[8]&(0xFF>>2) | (0x02 << 6)) // Set variant to RFC4122

	if id.IsNil() {
		return Nil, errors.New("new id is nil")
	}

	return id, nil
}

// ParseNillableV4 will attempt to create a new V4 UUID out of the given string.
// Nil UUIDs are not treated as errors.
func ParseNillableV4(id string) (V4, error) {
	if id != Nil.String() && !validV4.MatchString(id) {
		return Nil, errors.New("invalid uuid")
	}

	id = strings.ReplaceAll(id, "-", "")
	decoded, err := hex.DecodeString(id)
	if err != nil {
		return Nil, fmt.Errorf("decode hex id string: %w", err)
	}

	return V4(*(*[16]byte)(decoded)), nil
}

// ParseV4 will attempt to create a new V4 UUID out of the given string.
func ParseV4(id string) (V4, error) {
	if id == Nil.String() {
		return Nil, errors.New("invalid uuid")
	}

	return ParseNillableV4(id)
}

// ParseV4OrNil will always return a V4 UUID, returning Nil even on error.
func ParseV4OrNil(id string) V4 {
	v4, err := ParseNillableV4(id)
	if err != nil {
		return Nil
	}

	return v4
}

// IsNil checks whether the current V4 UUID is nil or not.
func (id V4) IsNil() bool {
	return id == Nil
}

// IsValid checks to see whether the current V4 UUID is in a valid state.
// The nil UUID is considered invalid.
func (id V4) IsValid() bool {
	return validV4.MatchString(id.String())
}

// String implements the Stringer interface.
// It will return the UUID in the format "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx".
func (id V4) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}

// Scan implements sql.Scanner to allow scanning UUIDs from SQL databases.
func (id *V4) Scan(src any) error {
	switch src := src.(type) {
	case nil:
		*id = Nil

	case string:
		u, err := ParseV4(src)
		if err != nil {
			return fmt.Errorf("parse v4 %T: %w", src, err)
		}

		*id = u

	case []byte:
		u, err := ParseV4(string(src))
		if err != nil {
			return fmt.Errorf("parse v4 %T: %w", src, err)
		}

		*id = u

	default:
		return fmt.Errorf("unable to scan %T into V4 UUID", src)
	}

	return nil
}

// Value implements sql.Valuer to allow using UUIDs directly with SQL databases.
func (id V4) Value() (driver.Value, error) {
	if id.IsNil() {
		return nil, nil
	}

	return id.String(), nil
}
