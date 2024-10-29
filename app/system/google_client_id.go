package system

import (
	"regexp"

	"github.com/polyscone/tofu/internal/i18n"
)

var validGoogleClientIDSeq = regexp.MustCompile(`^.+$`)

type GoogleClientID string

func NewGoogleClientID(id string) (GoogleClientID, error) {
	if id == "" {
		return "", nil
	}

	if !validGoogleClientIDSeq.MatchString(id) {
		return "", i18n.M("google_client_id.error.invalid")
	}

	return GoogleClientID(id), nil
}

func (e GoogleClientID) String() string {
	return string(e)
}
