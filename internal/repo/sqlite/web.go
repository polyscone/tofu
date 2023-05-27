package sqlite

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/fs"
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/repo"
)

const (
	webTokenKindActivation    = "activation"
	webTokenKindResetPassword = "reset_password"
)

type WebRepo struct {
	db *DB
}

func NewWebRepo(ctx context.Context, db *sql.DB, sessionLifespan time.Duration) (*WebRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/web")
	if err != nil {
		return nil, errors.Tracef(err)
	}

	if err := migrateFS(ctx, db, "web", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	// Background goroutine to clean up expired sessions
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(sessionLifespan) {
			_, err := db.ExecContext(ctx, `
				DELETE FROM web__sessions
				WHERE updated_at <= :expires_at;
			`,
				sql.Named("expires_at", Time(time.Now().Add(-sessionLifespan).UTC())),
			)
			if err != nil {
				logger.PrintError(err)
			}
		}
	})

	// Background goroutine to clean up expired tokens
	background.Go(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			_, err := db.ExecContext(ctx, `
				DELETE FROM web__tokens
				WHERE expires_at <= :expires_at;
			`,
				sql.Named("expires_at", Time(time.Now().UTC())),
			)
			if err != nil {
				logger.PrintError(err)
			}
		}
	})

	r := WebRepo{db: newDB(db)}

	return &r, nil
}

func (r *WebRepo) FindSessionDataByID(ctx context.Context, id string) (session.Data, error) {
	var data []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT data
		FROM web__sessions
		WHERE id = :id;
	`,
		sql.Named("id", id),
	).Scan(&data)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			err = errors.Tracef(err, session.ErrNotFound)
		}

		return nil, errors.Tracef(err)
	}

	d := json.NewDecoder(bytes.NewReader(data))

	d.UseNumber()

	var res session.Data
	err = d.Decode(&res)

	return res, errors.Tracef(err)
}

func (r *WebRepo) SaveSession(ctx context.Context, s session.Session) error {
	b, err := json.Marshal(s.Data)
	if err != nil {
		return errors.Tracef(err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO web__sessions
			(id, data, updated_at)
		VALUES
			(:id, :data, :updated_at);
	`,
		sql.Named("id", s.ID),
		sql.Named("data", b),
		sql.Named("updated_at", Time(time.Now().UTC())),
	)

	return errors.Tracef(err)
}

func (r *WebRepo) DestroySession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM web__sessions
		WHERE id = :id;
	`,
		sql.Named("id", id),
	)

	return errors.Tracef(err)
}

func (r *WebRepo) add(ctx context.Context, email string, ttl time.Duration, kind string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM web__tokens
		WHERE
			email = :email AND
			kind = :kind;
	`,
		sql.Named("email", email),
		sql.Named("kind", kind),
	)
	if err != nil {
		return "", errors.Tracef(err)
	}

	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Tracef(err)
	}

	b32 := base32.StdEncoding.WithPadding(base32.NoPadding)
	token := make([]byte, b32.EncodedLen(len(b)))
	b32.Encode(token, b)

	sum := sha256.Sum256(token)
	hash := sum[:]

	expiresAt := Time(time.Now().Add(ttl).UTC())

	_, err = tx.ExecContext(ctx, `
		INSERT INTO web__tokens (
			hash,
			email,
			kind,
			expires_at
		) VALUES (
			:hash,
			:email,
			:kind,
			:expires_at
		);
	`,
		sql.Named("hash", hash),
		sql.Named("email", email),
		sql.Named("kind", kind),
		sql.Named("expires_at", expiresAt),
	)
	if err != nil {
		return "", errors.Tracef(err)
	}

	return string(token), errors.Tracef(tx.Commit())
}

func (r *WebRepo) AddActivationToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	return r.add(ctx, email, ttl, webTokenKindActivation)
}

func (r *WebRepo) AddResetPasswordToken(ctx context.Context, email string, ttl time.Duration) (string, error) {
	return r.add(ctx, email, ttl, webTokenKindResetPassword)
}

func (r *WebRepo) findToken(ctx context.Context, token, kind string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var email string

	err := r.db.QueryRowContext(ctx, `
		SELECT email
		FROM web__tokens
		WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at;
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(time.Now().UTC())),
	).Scan(&email)

	return email, errors.Tracef(repoerr(err))
}

func (r *WebRepo) FindActivationTokenEmail(ctx context.Context, token string) (string, error) {
	return r.findToken(ctx, token, webTokenKindActivation)
}

func (r *WebRepo) FindResetPasswordTokenEmail(ctx context.Context, token string) (string, error) {
	return r.findToken(ctx, token, webTokenKindResetPassword)
}

func (r *WebRepo) consume(ctx context.Context, token, kind string) error {
	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	res, err := r.db.ExecContext(ctx, `
		DELETE FROM web__tokens	WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at;
	`,
		sql.Named("hash", hash),
		sql.Named("kind", kind),
		sql.Named("expires_at", Time(time.Now().UTC())),
	)
	if err != nil {
		return errors.Tracef(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Tracef(err)
	}
	if affected == 0 {
		return errors.Tracef(repo.ErrNotFound)
	}

	return nil
}

func (r *WebRepo) ConsumeActivationToken(ctx context.Context, token string) error {
	return r.consume(ctx, token, webTokenKindActivation)
}

func (r *WebRepo) ConsumeResetPasswordToken(ctx context.Context, token string) error {
	return r.consume(ctx, token, webTokenKindResetPassword)
}
