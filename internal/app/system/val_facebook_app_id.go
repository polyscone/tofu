package system

import (
	"errors"
	"regexp"
)

var validFacebookAppIDSeq = regexp.MustCompile(`^.+$`)

type FacebookAppID string

func NewFacebookAppID(id string) (FacebookAppID, error) {
	if id == "" {
		return "", nil
	}

	if !validFacebookAppIDSeq.MatchString(id) {
		return "", errors.New("invalid app id")
	}

	return FacebookAppID(id), nil
}

func (e FacebookAppID) String() string {
	return string(e)
}
