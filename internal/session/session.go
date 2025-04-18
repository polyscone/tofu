package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
)

// Data represents a sessions data as key/value pairs.
type Data map[string]any

type Status int

const (
	Unchanged Status = iota
	Accessed
	Modified
	Destroyed
	placeholder
)

type Session struct {
	mu         sync.RWMutex
	ID         string
	Status     Status
	data       Data
	originalID string
}

func newSessionPlaceholder() *Session {
	return &Session{Status: placeholder}
}

func newSession() (*Session, error) {
	id, err := newID()
	if err != nil {
		return &Session{}, fmt.Errorf("new id: %w", err)
	}

	s := &Session{ID: id}

	s.setStatus(Unchanged)

	return s, nil
}

func (s *Session) Data() Data {
	return s.data
}

func (s *Session) setData(data Data) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = data
}

func (s *Session) set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(Data)
	}

	s.data[key] = value
}

func (s *Session) get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data[key]
}

func (s *Session) has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]

	return ok
}

func (s *Session) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
}

func (s *Session) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.data)
}

func (s *Session) setStatus(status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
