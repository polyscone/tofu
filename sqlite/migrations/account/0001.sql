CREATE TABLE account__users (
	id                          TEXT NOT NULL PRIMARY KEY,
	email                       TEXT NOT NULL UNIQUE COLLATE NOCASE,
	hashed_password             BLOB,
	totp_method                 TEXT NOT NULL,
	totp_tel                    TEXT NOT NULL,
	totp_key                    BLOB,
	totp_algorithm              TEXT NOT NULL,
	totp_digits                 INTEGER NOT NULL,
	totp_period                 TEXT NOT NULL,
	totp_verified_at            TEXT,
	totp_activated_at           TEXT,
	invited_at                  TEXT,
	signed_up_at                TEXT,
	signed_up_system            TEXT NOT NULL,
	signed_up_method            TEXT NOT NULL,
	verified_at                 TEXT,
	activated_at                TEXT,
	last_sign_in_attempt_at     TEXT,
	last_sign_in_attempt_system TEXT NOT NULL,
	last_sign_in_attempt_method TEXT NOT NULL,
	last_signed_in_at           TEXT,
	last_signed_in_system       TEXT NOT NULL,
	last_signed_in_method       TEXT NOT NULL,
	suspended_at                TEXT,
	suspended_reason            TEXT NOT NULL,
	created_at                  TEXT NOT NULL,
	updated_at                  TEXT
) STRICT;

CREATE TABLE account__totp_reset_requests (
	user_id      TEXT NOT NULL PRIMARY KEY,
	requested_at TEXT,
	approved_at  TEXT,
	created_at   TEXT NOT NULL,
	updated_at   TEXT,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE
) STRICT;

CREATE TABLE account__sign_in_attempt_logs (
	email           TEXT NOT NULL PRIMARY KEY,
	attempts        INTEGER NOT NULL,
	last_attempt_at TEXT NOT NULL,
	created_at      TEXT NOT NULL,
	updated_at      TEXT
) STRICT;

CREATE TABLE account__roles (
	id          TEXT NOT NULL PRIMARY KEY,
	name        TEXT NOT NULL UNIQUE COLLATE NOCASE,
	description TEXT NOT NULL COLLATE NOCASE,
	created_at  TEXT NOT NULL,
	updated_at  TEXT
) STRICT;

CREATE TABLE account__permissions (
	id         TEXT NOT NULL PRIMARY KEY,
	name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
	created_at TEXT NOT NULL,
	updated_at TEXT
) STRICT;

CREATE TABLE account__role_permissions (
	role_id       TEXT NOT NULL,
	permission_id TEXT NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT,
	FOREIGN KEY (role_id) REFERENCES account__roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES account__permissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (role_id, permission_id)
) STRICT;

CREATE TABLE account__user_roles (
	user_id    TEXT NOT NULL,
	role_id    TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (role_id) REFERENCES account__roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, role_id)
) STRICT;

CREATE TABLE account__user_grants (
	user_id       TEXT NOT NULL,
	permission_id TEXT NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES account__permissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, permission_id)
) STRICT;

CREATE TABLE account__user_denials (
	user_id       TEXT NOT NULL,
	permission_id TEXT NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES account__permissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, permission_id)
) STRICT;

CREATE TABLE account__recovery_codes (
	user_id     TEXT NOT NULL,
	hashed_code BLOB NOT NULL,
	created_at  TEXT NOT NULL,
	updated_at  TEXT,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, hashed_code)
) STRICT;
