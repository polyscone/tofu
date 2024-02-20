package uuid

import (
	"crypto/rand"
	"fmt"
	"io"
	"regexp"
)

var validV4 = regexp.MustCompile("(?i)^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$")

func NewV4() (UUID, error) {
	var id UUID

	if _, err := io.ReadFull(rand.Reader, id[:]); err != nil {
		return Nil, fmt.Errorf("read random bytes: %w", err)
	}

	id[6] = (id[6] & 0x0F) | (0x04 << 4)    // Set version to 4
	id[8] = (id[8]&(0xFF>>2) | (0x02 << 6)) // Set variant to RFC4122

	return id, nil
}

func (id UUID) IsValidV4() bool {
	return validV4.MatchString(id.String())
}
