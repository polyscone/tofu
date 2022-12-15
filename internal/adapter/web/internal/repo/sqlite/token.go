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

func (r *TokenRepo) AddActivationToken(ctx context.Context, email text.Email, ttl time.Duration) (string, error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	stmt, args := "DELETE FROM web__tokens WHERE email = :email;", sqlite.Args{
		"email": email,
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
			(id, hash, email, expires_at)
		VALUES
			(:id, :hash, :email, :expires_at);
	`, sqlite.Args{
		"id":         id,
		"hash":       hash,
		"email":      email,
		"expires_at": expiresAt,
	}
	_, err = tx.Exec(ctx, stmt, args)

	return string(token), errors.Tracef(tx.Commit())
}

func (r *TokenRepo) Consume(ctx context.Context, token string) (text.Email, error) {
	tx, err := r.db.Begin(ctx, nil)
	if err != nil {
		return "", errors.Tracef(err)
	}
	defer tx.Rollback()

	sum := sha256.Sum256([]byte(token))
	hash := sum[:]

	var email text.Email

	stmt, args := `
		SELECT email
		FROM web__tokens
		WHERE
			hash = :hash AND
			expires_at > :expires_at;
	`, sqlite.Args{
		"hash":       hash,
		"expires_at": time.Now().UTC(),
	}
	err = tx.QueryRow(ctx, stmt, args).Scan(&email)
	if err != nil {
		return "", errors.Tracef(err)
	}

	stmt, args = "DELETE FROM web__tokens WHERE hash = :hash;", sqlite.Args{
		"hash": hash,
	}
	if _, err := tx.Exec(ctx, stmt, args); err != nil {
		return "", errors.Tracef(err)
	}

	return email, errors.Tracef(tx.Commit())
}
