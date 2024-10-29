package system

import (
	"regexp"

	"github.com/polyscone/tofu/internal/i18n"
)

var validFacebookAppIDSeq = regexp.MustCompile(`^.+$`)

type FacebookAppID string

func NewFacebookAppID(id string) (FacebookAppID, error) {
	if id == "" {
		return "", nil
	}

	if !validFacebookAppIDSeq.MatchString(id) {
		return "", i18n.M("facebook_app_id.error.invalid")
	}

	return FacebookAppID(id), nil
}

func (e FacebookAppID) String() string {
	return string(e)
}
