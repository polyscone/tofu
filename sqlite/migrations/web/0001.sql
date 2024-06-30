CREATE TABLE web__sessions (
	id         TEXT NOT NULL PRIMARY KEY,
	data       BLOB NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
) STRICT;

CREATE TABLE web__tokens (
	hash       BLOB NOT NULL PRIMARY KEY,
	value      TEXT NOT NULL,
	kind       TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT
) STRICT;
