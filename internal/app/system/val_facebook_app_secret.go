package system

import (
	"errors"
	"regexp"
)

var validFacebookAppSecretSeq = regexp.MustCompile(`^.+$`)

type FacebookAppSecret string

func NewFacebookAppSecret(secret string) (FacebookAppSecret, error) {
	if secret == "" {
		return "", nil
	}

	if !validFacebookAppSecretSeq.MatchString(secret) {
		return "", errors.New("invalid app secret")
	}

	return FacebookAppSecret(secret), nil
}

func (e FacebookAppSecret) String() string {
	return string(e)
}
