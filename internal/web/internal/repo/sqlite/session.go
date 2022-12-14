package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/session"
)

type SessionRepo struct {
	db *sqlite.DB
}

func NewSessionRepo(ctx context.Context, db *sqlite.DB, lifespan time.Duration) (*SessionRepo, error) {
	if err := db.MigrateFS(ctx, "web", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	// Background goroutine to clean up expired sessions
	background.Func(func() {
		ctx := context.Background()

		for range time.Tick(lifespan) {
			stmt, args := "DELETE FROM web__sessions WHERE updated_at <= :expires_at;", sqlite.Args{
				"expires_at": time.Now().Add(-lifespan).UTC(),
			}
			if _, err := db.Exec(ctx, stmt, args); err != nil {
				logger.PrintError(err)
			}
		}
	})

	return &SessionRepo{db: db}, nil
}

func (r *SessionRepo) FindByID(ctx context.Context, id string) (session.Data, error) {
	var data []byte

	stmt, args := "SELECT data FROM web__sessions WHERE id = :id;", sqlite.Args{"id": id}
	err := r.db.QueryRow(ctx, stmt, args).Scan(&data)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			err = errors.Tracef(err, session.ErrNotFound)
		}

		return nil, errors.Tracef(err)
	}

	var res session.Data
	err = json.Unmarshal(data, &res)

	return res, errors.Tracef(err)
}

func (r *SessionRepo) Save(ctx context.Context, s session.Session) error {
	b, err := json.Marshal(s.Data)
	if err != nil {
		return errors.Tracef(err)
	}

	stmt, args := `
		INSERT OR REPLACE INTO web__sessions
			(id, data, updated_at)
		VALUES
			(:id, :data, :updated_at);
	`, sqlite.Args{
		"id":         s.ID,
		"data":       b,
		"updated_at": time.Now().UTC(),
	}
	_, err = r.db.Exec(ctx, stmt, args)

	return errors.Tracef(err)
}

func (r *SessionRepo) Destroy(ctx context.Context, id string) error {
	stmt, args := "DELETE FROM web__sessions WHERE id = :id;", sqlite.Args{"id": id}
	_, err := r.db.Exec(ctx, stmt, args)

	return errors.Tracef(err)
}
