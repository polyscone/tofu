package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/system"
)

type SystemRepo struct {
	db *DB
}

func NewSystemRepo(ctx context.Context, db *DB) (*SystemRepo, error) {
	migrations, err := fs.Sub(migrations, "migrations/system")
	if err != nil {
		return nil, fmt.Errorf("initialise system migrations FS: %w", err)
	}

	if err := migrateFS(ctx, db, "system", migrations); err != nil {
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
	tx, err := r.db.BeginExclusiveTx(ctx, nil)
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
		select
			system_email,
			security_email,
			sign_up_enabled,
			sign_up_auto_activate_enabled,
			totp_required,
			totp_sms_enabled,
			magic_link_sign_in_enabled,
			google_sign_in_enabled,
			google_sign_in_client_id,
			facebook_sign_in_enabled,
			facebook_sign_in_app_id,
			facebook_sign_in_app_secret,
			resend_api_key,
			twilio_sid,
			twilio_token,
			twilio_from_tel
		from system__config
	`).Scan(
		&config.SystemEmail,
		&config.SecurityEmail,
		&config.SignUpEnabled,
		&config.SignUpAutoActivateEnabled,
		&config.TOTPRequired,
		&config.TOTPSMSEnabled,
		&config.MagicLinkSignInEnabled,
		&config.GoogleSignInEnabled,
		&config.GoogleSignInClientID,
		&config.FacebookSignInEnabled,
		&config.FacebookSignInAppID,
		&config.FacebookSignInAppSecret,
		&config.ResendAPIKey,
		&config.TwilioSID,
		&config.TwilioToken,
		&config.TwilioFromTel,
	)
	if err != nil && !errors.Is(err, app.ErrNotFound) {
		return nil, err
	}

	if errors.Is(err, app.ErrNotFound) {
		config.SetupRequired = true
		config.SignUpEnabled = true
		config.SignUpAutoActivateEnabled = true
	}

	return &config, nil
}

func (r *SystemRepo) upsertConfig(ctx context.Context, tx *Tx, config *system.Config) error {
	_, err := tx.ExecContext(ctx, `
		insert into system__config (
			id,
			system_email,
			security_email,
			sign_up_enabled,
			sign_up_auto_activate_enabled,
			totp_required,
			totp_sms_enabled,
			magic_link_sign_in_enabled,
			google_sign_in_enabled,
			google_sign_in_client_id,
			facebook_sign_in_enabled,
			facebook_sign_in_app_id,
			facebook_sign_in_app_secret,
			resend_api_key,
			twilio_sid,
			twilio_token,
			twilio_from_tel,
			created_at
		) values (
			:id,
			:system_email,
			:security_email,
			:sign_up_enabled,
			:sign_up_auto_activate_enabled,
			:totp_required,
			:totp_sms_enabled,
			:magic_link_sign_in_enabled,
			:google_sign_in_enabled,
			:google_sign_in_client_id,
			:facebook_sign_in_enabled,
			:facebook_sign_in_app_id,
			:facebook_sign_in_app_secret,
			:resend_api_key,
			:twilio_sid,
			:twilio_token,
			:twilio_from_tel,
			:created_at
		)
		on conflict (id) do
			update set
				system_email = excluded.system_email,
				security_email = excluded.security_email,
				sign_up_enabled = excluded.sign_up_enabled,
				sign_up_auto_activate_enabled = excluded.sign_up_auto_activate_enabled,
				totp_required = excluded.totp_required,
				totp_sms_enabled = excluded.totp_sms_enabled,
				magic_link_sign_in_enabled = excluded.magic_link_sign_in_enabled,
				google_sign_in_enabled = excluded.google_sign_in_enabled,
				google_sign_in_client_id = excluded.google_sign_in_client_id,
				facebook_sign_in_enabled = excluded.facebook_sign_in_enabled,
				facebook_sign_in_app_id = excluded.facebook_sign_in_app_id,
				facebook_sign_in_app_secret = excluded.facebook_sign_in_app_secret,
				resend_api_key = excluded.resend_api_key,
				twilio_sid = excluded.twilio_sid,
				twilio_token = excluded.twilio_token,
				twilio_from_tel = excluded.twilio_from_tel,
				updated_at = :updated_at
			where
				system_email != excluded.system_email or
				security_email != excluded.security_email or
				sign_up_enabled != excluded.sign_up_enabled or
				sign_up_auto_activate_enabled != excluded.sign_up_auto_activate_enabled or
				totp_required != excluded.totp_required or
				totp_sms_enabled != excluded.totp_sms_enabled or
				magic_link_sign_in_enabled != excluded.magic_link_sign_in_enabled or
				google_sign_in_enabled != excluded.google_sign_in_enabled or
				google_sign_in_client_id != excluded.google_sign_in_client_id or
				facebook_sign_in_enabled != excluded.facebook_sign_in_enabled or
				facebook_sign_in_app_id != excluded.facebook_sign_in_app_id or
				facebook_sign_in_app_secret != excluded.facebook_sign_in_app_secret or
				resend_api_key != excluded.resend_api_key or
				twilio_sid != excluded.twilio_sid or
				twilio_token != excluded.twilio_token or
				twilio_from_tel != excluded.twilio_from_tel
	`,
		sql.Named("id", 1),
		sql.Named("system_email", config.SystemEmail),
		sql.Named("security_email", config.SecurityEmail),
		sql.Named("sign_up_enabled", config.SignUpEnabled),
		sql.Named("sign_up_auto_activate_enabled", config.SignUpAutoActivateEnabled),
		sql.Named("totp_required", config.TOTPRequired),
		sql.Named("totp_sms_enabled", config.TOTPSMSEnabled),
		sql.Named("magic_link_sign_in_enabled", config.MagicLinkSignInEnabled),
		sql.Named("google_sign_in_enabled", config.GoogleSignInEnabled),
		sql.Named("google_sign_in_client_id", config.GoogleSignInClientID),
		sql.Named("facebook_sign_in_enabled", config.FacebookSignInEnabled),
		sql.Named("facebook_sign_in_app_id", config.FacebookSignInAppID),
		sql.Named("facebook_sign_in_app_secret", config.FacebookSignInAppSecret),
		sql.Named("resend_api_key", config.ResendAPIKey),
		sql.Named("twilio_sid", config.TwilioSID),
		sql.Named("twilio_token", config.TwilioToken),
		sql.Named("twilio_from_tel", config.TwilioFromTel),
		sql.Named("created_at", Time(tx.now.UTC())),
		sql.Named("updated_at", Time(tx.now.UTC())),
	)

	return err
}
