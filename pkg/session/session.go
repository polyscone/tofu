package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Status int

const (
	Unchanged Status = iota
	Accessed
	Modified
	Destroyed
	placeholder
)

type Session struct {
	ID         string
	Data       Data
	Status     Status
	originalID string
}

func newSessionPlaceholder() *Session {
	return &Session{Status: placeholder}
}

func newSession() (Session, error) {
	id, err := newID()
	if err != nil {
		return Session{}, fmt.Errorf("new id: %w", err)
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
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
