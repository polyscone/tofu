CREATE TABLE web__sessions (
  id         TEXT NOT NULL PRIMARY KEY,
  data       TEXT NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE web__tokens (
  hash       TEXT NOT NULL PRIMARY KEY,
  email      TEXT NOT NULL,
  kind       TEXT NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME
);
