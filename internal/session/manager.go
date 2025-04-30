package session

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type ctxKey int

const ctxSession ctxKey = iota

const managerKey = "__key"

var ErrNotFound = errors.New("not found")

// Reader defines the interface for reading session data.
type Reader interface {
	FindSessionDataByID(ctx context.Context, id string) (Data, error)
}

// Reader defines the interface for writing session data.
type Writer interface {
	SaveSession(ctx context.Context, s *Session) error
	DestroySession(ctx context.Context, id string) error
}

// ReadWriter is the combination of the Reader and Writer interfaces.
type ReadWriter interface {
	Reader
	Writer
}

// Manager loads, creates, and commits session data via contexts.
type Manager struct {
	mu   sync.RWMutex
	key  string
	repo ReadWriter
}

// NewManager creates a new session manager that will use the provided
// repository to interact with session data.
func NewManager(repo ReadWriter) (*Manager, error) {
	m := Manager{repo: repo}
	if err := m.RenewKey(); err != nil {
		return nil, fmt.Errorf("renew key: %w", err)
	}

	return &m, nil
}

// RenewKey replaces the session manager key with a new random one.
// Any sessions that have the incorrect key are automatically marked as being destroyed.
func (m *Manager) RenewKey() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("read random bytes: %w", err)
	}

	m.key = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(key)

	return nil
}

// Load attempts to load a session using the given id into the given context.
//
// If a session with the provided id is not found then a new session is
// created with a new id.
//
// After a session has been loaded or created it is set on the returned context.
func (m *Manager) Load(ctx context.Context, id string) (context.Context, error) {
	s := &Session{ID: id}

	data, err := m.repo.FindSessionDataByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		data = nil

		s, err = newSession()
		if err != nil {
			return ctx, fmt.Errorf("new session: %w", err)
		}

	case err != nil:
		return ctx, fmt.Errorf("find session data by id: %w", err)
	}

	m.mu.RLock()

	s.setData(data)
	s.set(managerKey, m.key)
	s.setStatus(Unchanged)

	m.mu.RUnlock()

	ctx = context.WithValue(ctx, ctxSession, s)

	return ctx, nil
}

// Commit uses the given context to save and/or destroy session data.
// Until Commit is called all session data only exists on a context.
func (m *Manager) Commit(ctx context.Context) (string, error) {
	s := getSession(ctx)

	m.mu.RLock()
	if key, _ := s.get(managerKey).(string); key != m.key {
		m.destroy(s)
	}
	m.mu.RUnlock()

	if s.originalID != "" {
		if err := m.repo.DestroySession(ctx, s.originalID); err != nil {
			return "", fmt.Errorf("destroy session: %w", err)
		}

		s.originalID = ""
	}

	switch s.Status {
	case Unchanged, Accessed, Modified:
		if err := m.repo.SaveSession(ctx, s); err != nil {
			return "", fmt.Errorf("save session: %w", err)
		}

	case Destroyed:
		if err := m.repo.DestroySession(ctx, s.ID); err != nil {
			return "", fmt.Errorf("destroy session: %w", err)
		}
	}

	return s.ID, nil
}

// Clear will empty a session's data and set its status to modified.
func (m *Manager) Clear(ctx context.Context) {
	s := getSession(ctx)

	s.clear()
	s.setStatus(Modified)
}

// Renew will update the id of the session on the given context.
// It will also track the original session id so it can be destroyed when
// session data is committed.
// Renewing a session's id will also set the session's status to modified.
func (m *Manager) Renew(ctx context.Context) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("new id: %w", err)
	}

	s := getSession(ctx)

	if s.originalID == "" {
		s.originalID = s.ID
	}

	s.ID = id

	s.setStatus(Modified)

	return nil
}

func (m *Manager) destroy(s *Session) {
	// If the session was renewed then original id will be populated
	// In this case we restore the original id to ensure it's destroyed
	// correctly in any session repository, since a renewal of a destroyed
	// session is meaningless anyway
	if s.originalID != "" {
		s.ID = s.originalID
		s.originalID = ""
	}

	s.setStatus(Destroyed)
}

// Destroy sets the status of the session on the given context to destroyed.
// Any persisted session data is not actually destroyed until any changes
// have been committed.
func (m *Manager) Destroy(ctx context.Context) {
	s := getSession(ctx)

	m.destroy(s)
}

// Status returns the status of the session on the given context.
func (m *Manager) Status(ctx context.Context) Status {
	s := getSession(ctx)

	return s.Status
}

// Set will set a key/value pair of session data in the session on the
// given context.
// Setting a value will also change the session's state to modified.
func (m *Manager) Set(ctx context.Context, key string, value any) {
	if t, ok := value.(time.Time); ok {
		value = t.Format(time.RFC3339Nano)
	}

	s := getSession(ctx)

	s.set(key, value)
	s.setStatus(Modified)
}

// Get will fetch the session value for the given key.
// If the key does not exist then the zero value is returned.
// Fetching data will set the session's status to accessed.
func (m *Manager) Get(ctx context.Context, key string) any {
	s := getSession(ctx)

	s.setStatus(Accessed)

	return s.get(key)
}

// Delete will delete the session value at the given key.
// Deleting data will also set the session's status to modified.
func (m *Manager) Delete(ctx context.Context, key string) {
	s := getSession(ctx)

	s.delete(key)
	s.setStatus(Modified)
}

// Has checks to see whether the given key exists in the session's data.
func (m *Manager) Has(ctx context.Context, key string) bool {
	s := getSession(ctx)

	return s.has(key)
}

// GetBool fetches a bool from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetBool(ctx context.Context, key string) bool {
	value, _ := m.Get(ctx, key).(bool)

	return value
}

// PopBool fetches a bool from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopBool(ctx context.Context, key string) bool {
	value := m.GetBool(ctx, key)

	m.Delete(ctx, key)

	return value
}

// GetInt fetches a int from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetInt(ctx context.Context, key string) int {
	switch value := m.Get(ctx, key).(type) {
	case int:
		return value

	case float64:
		if value == float64(int(value)) {
			return int(value)
		}

		return 0

	case json.Number:
		i, _ := value.Int64()

		return int(i)

	default:
		return 0
	}
}

// PopInt fetches a int from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopInt(ctx context.Context, key string) int {
	value := m.GetInt(ctx, key)

	m.Delete(ctx, key)

	return value
}

// GetFloat32 fetches a float32 from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetFloat32(ctx context.Context, key string) float32 {
	switch value := m.Get(ctx, key).(type) {
	case float32:
		return value

	case float64:
		return float32(value)

	case json.Number:
		i, _ := value.Float64()

		return float32(i)

	default:
		return 0
	}
}

// PopFloat32 fetches a float32 from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopFloat32(ctx context.Context, key string) float32 {
	value := m.GetFloat32(ctx, key)

	m.Delete(ctx, key)

	return value
}

// GetFloat64 fetches a float64 from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetFloat64(ctx context.Context, key string) float64 {
	switch value := m.Get(ctx, key).(type) {
	case float64:
		return value

	case json.Number:
		i, _ := value.Float64()

		return i

	default:
		return 0
	}
}

// PopFloat64 fetches a float64 from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopFloat64(ctx context.Context, key string) float64 {
	value := m.GetFloat64(ctx, key)

	m.Delete(ctx, key)

	return value
}

// GetString fetches a string from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetString(ctx context.Context, key string) string {
	value, _ := m.Get(ctx, key).(string)

	return value
}

// PopString fetches a string from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopString(ctx context.Context, key string) string {
	value := m.GetString(ctx, key)

	m.Delete(ctx, key)

	return value
}

// GetString fetches a string slice from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetStrings(ctx context.Context, key string) []string {
	switch slice := m.Get(ctx, key).(type) {
	case []string:
		return slice

	case []any:
		value := make([]string, len(slice))
		for i, v := range slice {
			if s, ok := v.(string); ok {
				value[i] = s
			}
		}

		return value

	default:
		return nil
	}
}

// PopString fetches a string slice from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopStrings(ctx context.Context, key string) []string {
	value := m.GetStrings(ctx, key)

	m.Delete(ctx, key)

	return value
}

// GetTime fetches a time.Time from the session's data.
// If the given key does not exist then the zero value is returned.
func (m *Manager) GetTime(ctx context.Context, key string) time.Time {
	value, _ := m.Get(ctx, key).(string)
	t, _ := time.Parse(time.RFC3339Nano, value)

	return t
}

// PopTime fetches a time.Time from the session's data.
// After fetching the value it is then deleted from the session.
// If the given key does not exist then the zero value is returned.
func (m *Manager) PopTime(ctx context.Context, key string) time.Time {
	value := m.GetTime(ctx, key)

	m.Delete(ctx, key)

	return value
}

func getSession(ctx context.Context) *Session {
	value := ctx.Value(ctxSession)
	if value == nil {
		return newSessionPlaceholder()
	}

	s, ok := value.(*Session)
	if !ok {
		panic(fmt.Sprintf("could not assert session as %T", s))
	}

	return s
}
