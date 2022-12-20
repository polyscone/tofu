package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"io"
	"time"

	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

const (
	activation    = "activation"
	resetPassword = "reset_password"
)

type TokenRepo struct {
	db *sqlite.DB
}

func NewTokenRepo(ctx context.Context, db *sqlite.DB) (*TokenRepo, error) {
	if err := db.MigrateFS(ctx, "web", migrations); err != nil {
		return nil, errors.Tracef(err)
	}

	// Background goroutine to clean up expired tokens
	background.Func(func() {
		ctx := context.Background()

		for range time.Tick(5 * time.Minute) {
			stmt, args := "DELETE FROM web__tokens WHERE expires_at <= :expires_at;", sqlite.Args{
				"expires_at": time.Now().UTC(),
			}
			if _, err := db.Exec(ctx, stmt, args); err != nil {
				logger.PrintError(err)
			}
		}
	})

	return &TokenRepo{db: db}, nil
}

func (r *TokenRepo) add(ctx context.Context, email text.Email, ttl time.Duration, kind string) (string, error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	stmt, args := `
		DELETE FROM web__tokens
		WHERE
			email = :email AND
			kind = :kind;
	`, sqlite.Args{
		"email": email,
		"kind":  kind,
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return "", errors.Tracef(err)
	}

	id, err := uuid.NewV4()
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

	expiresAt := time.Now().Add(ttl).UTC()

	stmt, args = `
		INSERT INTO web__tokens
			(id, hash, email, kind, expires_at)
		VALUES
			(:id, :hash, :email, :kind, :expires_at);
	`, sqlite.Args{
		"id":         id,
		"hash":       hash,
		"email":      email,
		"kind":       kind,
		"expires_at": expiresAt,
	}
	_, err = tx.Exec(ctx, stmt, args)

	return string(token), errors.Tracef(tx.Commit())
}

func (r *TokenRepo) AddActivationToken(ctx context.Context, email text.Email, ttl time.Duration) (string, error) {
	return r.add(ctx, email, ttl, activation)
}

func (r *TokenRepo) AddResetPasswordToken(ctx context.Context, email text.Email, ttl time.Duration) (string, error) {
	return r.add(ctx, email, ttl, resetPassword)
}

func (r *TokenRepo) consume(ctx context.Context, token, kind string) (text.Email, error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var id uuid.V4
	var email text.Email

	stmt, args := `
		SELECT id, email
		FROM web__tokens
		WHERE
			hash = :hash AND
			kind = :kind AND
			expires_at > :expires_at;
	`, sqlite.Args{
		"hash":       hash,
		"kind":       kind,
		"expires_at": time.Now().UTC(),
	}
	err = tx.QueryRow(ctx, stmt, args).Scan(&id, &email)
	if err != nil {
		return "", errors.Tracef(err)
	}

	stmt, args = "DELETE FROM web__tokens WHERE id = :id;", sqlite.Args{
		"id": id,
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return "", errors.Tracef(err)
	}

	return email, errors.Tracef(tx.Commit())
}

func (r *TokenRepo) ConsumeActivationToken(ctx context.Context, token string) (text.Email, error) {
	return r.consume(ctx, token, activation)
}

func (r *TokenRepo) ConsumeResetPasswordToken(ctx context.Context, token string) (text.Email, error) {
	return r.consume(ctx, token, resetPassword)
}
