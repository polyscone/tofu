package system

import (
	"regexp"

	"github.com/polyscone/tofu/internal/i18n"
)

var validFacebookAppSecretSeq = regexp.MustCompile(`^.+$`)

type FacebookAppSecret string

func NewFacebookAppSecret(secret string) (FacebookAppSecret, error) {
	if secret == "" {
		return "", nil
	}

	if !validFacebookAppSecretSeq.MatchString(secret) {
		return "", i18n.M("facebook_app_secret.error.invalid")
	}

	return FacebookAppSecret(secret), nil
}

func (e FacebookAppSecret) String() string {
	return string(e)
}
