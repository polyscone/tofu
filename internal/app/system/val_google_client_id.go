package system

import (
	"errors"
	"regexp"
)

var validGoogleClientID = regexp.MustCompile(`^.+`)

type GoogleClientID string

func NewGoogleClientID(id string) (GoogleClientID, error) {
	if id == "" {
		return "", nil
	}

	if !validGoogleClientID.MatchString(id) {
		return "", errors.New("invalid client id")
	}

	return GoogleClientID(id), nil
}

func (e GoogleClientID) String() string {
	return string(e)
}
