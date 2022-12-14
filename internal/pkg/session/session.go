package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

type Status int

const (
	Unchanged Status = iota
	Accessed
	Modified
	Destroyed
)

type Session struct {
	ID         string
	Data       Data
	Status     Status
	originalID string
}

func newSession() (Session, error) {
	id, err := newID()
	if err != nil {
		return Session{}, errors.Tracef(err)
	}

	s := Session{ID: id}

	s.setStatus(Unchanged)

	return s, nil
}

func (s *Session) setStatus(status Status) {
	switch s.Status {
	case Unchanged:
		s.Status = status

	case Accessed:
		if status == Modified || status == Destroyed {
			s.Status = status
		}

	case Modified:
		if status == Destroyed {
			s.Status = status
		}

	case Destroyed:
		// Do nothing
		// If a session is destroyed it should remain destroyed

	default:
		s.Status = status
	}
}

func newID() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Tracef(err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
