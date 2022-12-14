package repotest

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

func RunSessionTests(t *testing.T, sessions session.Repo) {
	ctx := context.Background()
	s := session.Session{ID: errors.Must(uuid.NewV4()).String()}

	// A session that hasn't been saved yet shouldn't be found
	// It needs to be both repo.ErrNotFound for normal repo use, and
	// session.ErrNotFound for use with the session manager
	_, err := sessions.FindByID(ctx, s.ID)
	if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
		t.Errorf("want repo.ErrNotFound; got %q", got)
	}
	if want, got := session.ErrNotFound, err; !errors.Is(got, want) {
		t.Errorf("want session.ErrNotFound; got %q", got)
	}

	// Saving should succeed
	if err := sessions.Save(ctx, s); err != nil {
		t.Errorf("want <nil>; got %q", err)
	}

	// Finding the saved session should succeed
	if _, err := sessions.FindByID(ctx, s.ID); err != nil {
		t.Errorf("want <nil>; got %q", err)
	}

	// Re-saving the session should succeed
	if err := sessions.Save(ctx, s); err != nil {
		t.Errorf("want <nil>; got %q", err)
	}

	// Destroying the session should succeed
	if err := sessions.Destroy(ctx, s.ID); err != nil {
		t.Errorf("want <nil>; got %q", err)
	}

	// A destroyed sessions should not be found
	// It needs to be both repo.ErrNotFound for normal repo use, and
	// session.ErrNotFound for use with the session manager
	_, err = sessions.FindByID(ctx, s.ID)
	if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
		t.Errorf("want repo.ErrNotFound; got %q", got)
	}
	if want, got := session.ErrNotFound, err; !errors.Is(got, want) {
		t.Errorf("want session.ErrNotFound; got %q", got)
	}
}
