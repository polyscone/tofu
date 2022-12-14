CREATE TABLE web__sessions (
  id         TEXT NOT NULL PRIMARY KEY,
  data       TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE web__tokens (
  id         TEXT NOT NULL PRIMARY KEY,
  hash       TEXT NOT NULL UNIQUE,
  email      TEXT NOT NULL UNIQUE,
  expires_at DATETIME NOT NULL
);
