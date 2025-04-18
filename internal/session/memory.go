package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// JSONMemoryRepo implements an in-memory repo for use with a session manager.
// The intended use of this repo is developer tests.
//
// We want to test against JSON here to make sure we handle numbers correctly,
// which is why we map to a byte slice
type JSONMemoryRepo struct {
	useNumber bool
	data      map[string][]byte
}

// NewJSONMemoryRepo returns a new in-memory session repo, intended
// for use in developer tests.
func NewJSONMemoryRepo(useNumber bool) *JSONMemoryRepo {
	return &JSONMemoryRepo{
		useNumber: useNumber,
		data:      make(map[string][]byte),
	}
}

// FindByID attempts to find session data using the given id.
func (r *JSONMemoryRepo) FindSessionDataByID(ctx context.Context, id string) (Data, error) {
	if data, ok := r.data[id]; ok {
		d := json.NewDecoder(bytes.NewReader(data))

		if r.useNumber {
			d.UseNumber()
		}

		var res Data
		err := d.Decode(&res)

		return res, err
	}

	return nil, ErrNotFound
}

// Save persists the given session in-memory.
func (r *JSONMemoryRepo) SaveSession(ctx context.Context, s *Session) error {
	b, err := json.Marshal(s.Data())
	if err != nil {
		return fmt.Errorf("marshal session data: %w", err)
	}

	r.data[s.ID] = b

	return nil
}

// Destroy deletes a session by the given id from memory.
func (r *JSONMemoryRepo) DestroySession(ctx context.Context, id string) error {
	delete(r.data, id)

	return nil
}
