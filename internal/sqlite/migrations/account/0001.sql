CREATE TABLE account__users (
	id                          TEXT NOT NULL PRIMARY KEY,
	email                       TEXT NOT NULL UNIQUE COLLATE NOCASE,
	hashed_password             TEXT,
	totp_method                 TEXT NOT NULL,
	totp_tel                    TEXT NOT NULL,
	totp_key                    TEXT,
	totp_algorithm              TEXT NOT NULL,
	totp_digits                 INTEGER NOT NULL,
	totp_period                 TEXT NOT NULL,
	totp_verified_at            DATETIME,
	totp_activated_at           DATETIME,
	invited_at                  DATETIME,
	signed_up_at                DATETIME,
	signed_up_system            TEXT NOT NULL,
	signed_up_method            TEXT NOT NULL,
	verified_at                 DATETIME,
	activated_at                DATETIME,
	last_sign_in_attempt_at     DATETIME,
	last_sign_in_attempt_system TEXT NOT NULL,
	last_sign_in_attempt_method TEXT NOT NULL,
	last_signed_in_at           DATETIME,
	last_signed_in_system       TEXT NOT NULL,
	last_signed_in_method       TEXT NOT NULL,
	suspended_at                DATETIME,
	suspended_reason            TEXT NOT NULL,
	created_at                  DATETIME NOT NULL,
	updated_at                  DATETIME
);
CREATE INDEX idx_account__users_email ON account__users(email);

CREATE TABLE account__totp_reset_requests (
	user_id      TEXT NOT NULL PRIMARY KEY,
	requested_at DATETIME,
	approved_at  DATETIME,
	created_at   DATETIME NOT NULL,
	updated_at   DATETIME,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE account__sign_in_attempt_logs (
	email           TEXT NOT NULL PRIMARY KEY,
	attempts        INTEGER NOT NULL,
	last_attempt_at DATETIME NOT NULL,
	created_at      DATETIME NOT NULL,
	updated_at      DATETIME
);

CREATE TABLE account__roles (
	id          TEXT NOT NULL PRIMARY KEY,
	name        TEXT NOT NULL UNIQUE COLLATE NOCASE,
	description TEXT NOT NULL COLLATE NOCASE,
	created_at  DATETIME NOT NULL,
	updated_at  DATETIME
);

CREATE TABLE account__permissions (
	id         TEXT NOT NULL PRIMARY KEY,
	name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
	created_at DATETIME NOT NULL,
	updated_at DATETIME
);

CREATE TABLE account__role_permissions (
	role_id       TEXT NOT NULL,
	permission_id TEXT NOT NULL,
	created_at    DATETIME NOT NULL,
	updated_at    DATETIME,
	FOREIGN KEY (role_id) REFERENCES account__roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES account__permissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE account__user_roles (
	user_id    TEXT NOT NULL,
	role_id    TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (role_id) REFERENCES account__roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

CREATE TABLE account__user_grants (
	user_id       TEXT NOT NULL,
	permission_id TEXT NOT NULL,
	created_at    DATETIME NOT NULL,
	updated_at    DATETIME,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES account__permissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, permission_id)
);

CREATE TABLE account__user_denials (
	user_id       TEXT NOT NULL,
	permission_id TEXT NOT NULL,
	created_at    DATETIME NOT NULL,
	updated_at    DATETIME,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES account__permissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, permission_id)
);

CREATE TABLE account__recovery_codes (
	user_id     TEXT NOT NULL,
	hashed_code TEXT NOT NULL,
	created_at  DATETIME NOT NULL,
	updated_at  DATETIME,
	FOREIGN KEY (user_id) REFERENCES account__users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY (user_id, hashed_code)
);