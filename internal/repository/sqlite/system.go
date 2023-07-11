package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/repository"
)

type SystemRepo struct {
	db *DB
}

func NewSystemRepo(ctx context.Context, db *DB) (*SystemRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/system")
	if err != nil {
		return nil, fmt.Errorf("initialise system migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db.DB, "system", migrations); err != nil {
		return nil, fmt.Errorf("migrate system: %w", err)
	}

	r := SystemRepo{db: db}

	return &r, nil
}

func (r *SystemRepo) FindConfig(ctx context.Context) (*system.Config, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	return r.findConfig(ctx, tx)
}

func (r *SystemRepo) SaveConfig(ctx context.Context, config *system.Config) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.upsertConfig(ctx, tx, config); err != nil {
		return fmt.Errorf("upsert config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func (r *SystemRepo) findConfig(ctx context.Context, tx *Tx) (*system.Config, error) {
	var config system.Config
	err := tx.QueryRowContext(ctx, `
		SELECT
			system_email,
			security_email,
			sign_up_enabled,
			totp_required,
			google_sign_in_enabled,
			google_sign_in_client_id,
			twilio_sid,
			twilio_token,
			twilio_from_tel
		FROM system__config
	`).Scan(
		&config.SystemEmail,
		&config.SecurityEmail,
		&config.SignUpEnabled,
		&config.TOTPRequired,
		&config.GoogleSignInEnabled,
		&config.GoogleSignInClientID,
		&config.TwilioSID,
		&config.TwilioToken,
		&config.TwilioFromTel,
	)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	if errors.Is(err, repository.ErrNotFound) {
		config.SetupRequired = true
		config.SignUpEnabled = true
	}

	return &config, nil
}

func (r *SystemRepo) upsertConfig(ctx context.Context, tx *Tx, config *system.Config) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO system__config (
			id,
			system_email,
			security_email,
			sign_up_enabled,
			totp_required,
			google_sign_in_enabled,
			google_sign_in_client_id,
			twilio_sid,
			twilio_token,
			twilio_from_tel,
			created_at
		) VALUES (
			:id,
			:system_email,
			:security_email,
			:sign_up_enabled,
			:totp_required,
			:google_sign_in_enabled,
			:google_sign_in_client_id,
			:twilio_sid,
			:twilio_token,
			:twilio_from_tel,
			:created_at
		)
		ON CONFLICT DO
			UPDATE SET
				system_email = :system_email,
				security_email = :security_email,
				sign_up_enabled = :sign_up_enabled,
				totp_required = :totp_required,
				google_sign_in_enabled = :google_sign_in_enabled,
				google_sign_in_client_id = :google_sign_in_client_id,
				twilio_sid = :twilio_sid,
				twilio_token = :twilio_token,
				twilio_from_tel = :twilio_from_tel,
				updated_at = :updated_at
	`,
		sql.Named("id", 1),
		sql.Named("system_email", config.SystemEmail),
		sql.Named("security_email", config.SecurityEmail),
		sql.Named("sign_up_enabled", config.SignUpEnabled),
		sql.Named("totp_required", config.TOTPRequired),
		sql.Named("google_sign_in_enabled", config.GoogleSignInEnabled),
		sql.Named("google_sign_in_client_id", config.GoogleSignInClientID),
		sql.Named("twilio_sid", config.TwilioSID),
		sql.Named("twilio_token", config.TwilioToken),
		sql.Named("twilio_from_tel", config.TwilioFromTel),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)

	return err
}
