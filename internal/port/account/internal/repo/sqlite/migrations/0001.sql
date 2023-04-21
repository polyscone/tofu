CREATE TABLE account__users (
  id               TEXT NOT NULL PRIMARY KEY,
  email            TEXT UNIQUE COLLATE NOCASE,
  hashed_password  TEXT NOT NULL,
  totp_key         TEXT NOT NULL,
  totp_verified_at DATETIME,
  activated_at     DATETIME,
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_account__users_email ON account__users (email);

CREATE TABLE account__roles (
  id         TEXT NOT NULL PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE account__permissions (
  id         TEXT NOT NULL PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP

);

CREATE TABLE account__role_permissions (
  role_id       TEXT NOT NULL,
  permission_id TEXT NOT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (role_id) REFERENCES account__roles (id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY (permission_id) REFERENCES account__permissions (id) ON DELETE CASCADE ON UPDATE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE account__user_roles (
  user_id    TEXT NOT NULL,
  role_id    TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES account__users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY (role_id) REFERENCES account__roles (id) ON DELETE CASCADE ON UPDATE CASCADE,
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE account__recovery_codes (
  user_id    TEXT NOT NULL,
  code       TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES account__users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  PRIMARY KEY (user_id, code)
);
