package session

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

type ctxKey int

const ctxSession ctxKey = iota

// Data represents a sessions data as key/value pairs.
type Data map[string]any

var ErrNotFound = errors.New("not found")

// Repo represents the interface required by a session manager to
// work with session data.
type Repo interface {
	FindByID(ctx context.Context, id string) (Data, error)
	Save(ctx context.Context, s Session) error
	Destroy(ctx context.Context, id string) error
}

// Manager loads, creates, and commits session data via contexts.
type Manager struct {
	repo Repo
}

// NewManager creates a new session manager that will use the provided
// repository to interact with session data.
func NewManager(repo Repo) *Manager {
	return &Manager{repo: repo}
}

// Load attempts to load a session using the given id into the given context.
//
// If a session with the provided id is not found then a new session is
// created with a new id.
//
// After a session has been loaded or created it is set on the returned context.
func (m *Manager) Load(ctx context.Context, id string) (context.Context, error) {
	s := Session{ID: id}

	data, err := m.repo.FindByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		data = nil

		s, err = newSession()
		if err != nil {
			return ctx, errors.Tracef(err)
		}

	case err != nil:
		return ctx, errors.Tracef(err)
	}

	s.Data = data

	s.setStatus(Unchanged)

	ctx = context.WithValue(ctx, ctxSession, &s)

	return ctx, nil
}

// Commit uses the given context to save and/or destroy session data.
// Until Commit is called all session data only exists on a context.
func (m *Manager) Commit(ctx context.Context) (string, error) {
	s := getSession(ctx)

	if s.originalID != "" {
		if err := m.repo.Destroy(ctx, s.originalID); err != nil {
			return "", errors.Tracef(err)
		}

		s.originalID = ""
	}

	switch s.Status {
	case Unchanged, Accessed, Modified:
		if err := m.repo.Save(ctx, *s); err != nil {
			return "", errors.Tracef(err)
		}

	case Destroyed:
		if err := m.repo.Destroy(ctx, s.ID); err != nil {
			return "", errors.Tracef(err)
		}
	}

	return s.ID, nil
}

// Renew will update the id of the session on the given context.
// It will also track the original session id so it can be destroyed when
// session data is committed.
// Renewing a session's id will also set the session's status to modified.
func (m *Manager) Renew(ctx context.Context) error {
	id, err := newID()
	if err != nil {
		return errors.Tracef(err)
	}

	s := getSession(ctx)

	if s.originalID == "" {
		s.originalID = s.ID
	}

	s.ID = id

	s.setStatus(Modified)

	return nil
}

// Destroy sets the status of the session on the given context to destroyed.
// Any persisted session data is not actually destroyed until any changes
// have been committed.
func (m *Manager) Destroy(ctx context.Context) {
	s := getSession(ctx)

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

// Status returns the status of the session on the given context.
func (m *Manager) Status(ctx context.Context) Status {
	s := getSession(ctx)

	return s.Status
}

// Set will set a key/value pair of session data in the session on the
// given context.
// Setting a value will also change the session's state to modified.
func (m *Manager) Set(ctx context.Context, key string, value any) {
	s := getSession(ctx)
	if s.Data == nil {
		s.Data = make(Data)
	}

	s.Data[key] = value

	s.setStatus(Modified)
}

// Get will fetch the session value for the given key.
// If the key does not exist then the zero value is returned.
// Fetching data will set the session's status to accessed.
func (m *Manager) Get(ctx context.Context, key string) any {
	s := getSession(ctx)

	s.setStatus(Accessed)

	return s.Data[key]
}

// Delete will delete the session value at the given key.
// Deleting data will also set the session's status to modified.
func (m *Manager) Delete(ctx context.Context, key string) {
	s := getSession(ctx)

	delete(s.Data, key)

	s.setStatus(Modified)
}

// Has checks to see whether the given key exists in the session's data.
func (m *Manager) Has(ctx context.Context, key string) bool {
	s := getSession(ctx)

	_, ok := s.Data[key]

	return ok
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
	value, _ := m.Get(ctx, key).(int)

	return value
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
	value, _ := m.Get(ctx, key).(float32)

	return value
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
	value, _ := m.Get(ctx, key).(float64)

	return value
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
