CREATE TABLE account__users (
  id                    INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  email                 TEXT NOT NULL UNIQUE COLLATE NOCASE,
  hashed_password       TEXT,
  totp_method           TEXT NOT NULL,
  totp_tel              TEXT NOT NULL,
  totp_key              TEXT,
  totp_algorithm        TEXT NOT NULL,
  totp_digits           INTEGER NOT NULL,
  totp_period_ns        INTEGER NOT NULL,
  totp_verified_at      DATETIME,
  totp_activated_at     DATETIME,
  signed_up_at          DATETIME NOT NULL,
  activated_at          DATETIME,
  last_signed_in_at     DATETIME,
  last_signed_in_method TEXT NOT NULL,
  created_at            DATETIME NOT NULL,
  updated_at            DATETIME
);
CREATE INDEX idx_account__users_email ON account__users (email);

CREATE TABLE account__roles (
  id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  name        TEXT NOT NULL UNIQUE COLLATE NOCASE,
  description TEXT NOT NULL,
  created_at  DATETIME NOT NULL,
  updated_at  DATETIME
);

CREATE TABLE account__permissions (
  id         INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME
);

CREATE TABLE account__role_permissions (
  role_id       INTEGER NOT NULL,
  permission_id INTEGER NOT NULL,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME,
  FOREIGN KEY (role_id) REFERENCES account__roles (id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY (permission_id) REFERENCES account__permissions (id) ON DELETE CASCADE ON UPDATE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE account__user_roles (
  user_id    INTEGER NOT NULL,
  role_id    INTEGER NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME,
  FOREIGN KEY (user_id) REFERENCES account__users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY (role_id) REFERENCES account__roles (id) ON DELETE CASCADE ON UPDATE CASCADE,
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE account__recovery_codes (
  user_id    INTEGER NOT NULL,
  code       TEXT NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME,
  FOREIGN KEY (user_id) REFERENCES account__users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  PRIMARY KEY (user_id, code)
);
