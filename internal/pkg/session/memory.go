package session

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

// MemoryRepo implements an in-memory repo for use with a session manager.
// The intended use of this repo is developer tests.
type MemoryRepo struct {
	data map[string]Data
}

// NewMemoryRepo returns a new in-memory session repo, intended
// for use in developer tests.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{data: make(map[string]Data)}
}

// FindByID attempts to find session data using the given id.
func (r *MemoryRepo) FindByID(ctx context.Context, id string) (Data, error) {
	if data, ok := r.data[id]; ok {
		return data, nil
	}

	return nil, errors.Tracef(ErrNotFound)
}

// Save persists the given session in-memory.
func (r *MemoryRepo) Save(ctx context.Context, s Session) error {
	r.data[s.ID] = s.Data

	return nil
}

// Destroy deletes a session by the given id from memory.
func (r *MemoryRepo) Destroy(ctx context.Context, id string) error {
	delete(r.data, id)

	return nil
}
